package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shuyahonda/crm-cli/internal/config"
)

// TokenResponse represents the Azure AD OAuth2 token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// DeviceCodeResponse represents the device code flow initiation response.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// Provider manages OAuth2 tokens with automatic refresh.
type Provider struct {
	cfg         *config.Config
	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// NewProvider creates a new auth provider.
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{cfg: cfg}
}

// GetToken returns a valid access token, refreshing if necessary.
func (p *Provider) GetToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return cached token if still valid (with 60s buffer)
	if p.token != "" && time.Now().Add(60*time.Second).Before(p.tokenExpiry) {
		return p.token, nil
	}

	switch p.cfg.AuthFlow {
	case "client_credentials":
		return p.fetchClientCredentialsToken(ctx)
	case "device_code":
		return p.fetchDeviceCodeToken(ctx)
	default:
		return "", fmt.Errorf("unsupported auth flow: %s", p.cfg.AuthFlow)
	}
}

// fetchClientCredentialsToken fetches a token using the client credentials flow.
func (p *Provider) fetchClientCredentialsToken(ctx context.Context) (string, error) {
	tokenURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
		p.cfg.TenantID,
	)

	// Dynamics 365 scope uses the CRM base URL
	scope := fmt.Sprintf("%s/.default", strings.TrimRight(p.cfg.CRMBaseURL, "/"))

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"scope":         {scope},
	}

	resp, err := postForm(ctx, tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}

	p.token = resp.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	return p.token, nil
}

// fetchDeviceCodeToken fetches a token using the device code flow (interactive).
func (p *Provider) fetchDeviceCodeToken(ctx context.Context) (string, error) {
	deviceCodeURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode",
		p.cfg.TenantID,
	)
	tokenURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/token",
		p.cfg.TenantID,
	)

	scope := fmt.Sprintf("%s/user_impersonation openid profile", strings.TrimRight(p.cfg.CRMBaseURL, "/"))

	// Step 1: Request device code
	deviceResp, err := http.PostForm(deviceCodeURL, url.Values{
		"client_id": {p.cfg.ClientID},
		"scope":     {scope},
	})
	if err != nil {
		return "", fmt.Errorf("device code request failed: %w", err)
	}
	defer deviceResp.Body.Close()

	body, _ := io.ReadAll(deviceResp.Body)
	var dcResp DeviceCodeResponse
	if err := json.Unmarshal(body, &dcResp); err != nil {
		return "", fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Show user the code
	fmt.Fprintf(os.Stderr, "\n%s\n\n", dcResp.Message)

	// Step 2: Poll for token
	interval := time.Duration(dcResp.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(dcResp.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}

		resp, err := postForm(ctx, tokenURL, url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {p.cfg.ClientID},
			"device_code": {dcResp.DeviceCode},
		})
		if err != nil {
			// "authorization_pending" is expected during polling
			continue
		}

		p.token = resp.AccessToken
		p.tokenExpiry = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
		return p.token, nil
	}

	return "", fmt.Errorf("device code flow timed out")
}

// postForm sends a POST request with form data and parses the token response.
func postForm(ctx context.Context, tokenURL string, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Check for OAuth error
	var errResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		if errResp.Error == "authorization_pending" || errResp.Error == "slow_down" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("OAuth error %s: %s", errResp.Error, errResp.ErrorDescription)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}
