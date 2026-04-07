// Package auth handles Azure AD / Entra ID authentication for Dynamics 365.
// It uses the official Microsoft Authentication Library (MSAL) for Go:
// https://github.com/AzureAD/microsoft-authentication-library-for-go
//
// Supported flows:
//   - device_code        — browser login, no app registration required
//   - password (ROPC)    — username/password, no app registration required
//   - client_credentials — service account, requires app registration
package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"

	"github.com/shuyahonda/crm-cli/internal/config"
)

// Provider acquires and caches OAuth2 tokens via MSAL.
type Provider struct {
	cfg   *config.Config
	cache *fileCache
}

// NewProvider creates a new MSAL-backed auth provider.
func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		cfg:   cfg,
		cache: newFileCache(),
	}
}

// GetToken returns a valid access token for Dynamics 365.
// Tokens are cached on disk; silent acquisition is attempted first.
func (p *Provider) GetToken(ctx context.Context) (string, error) {
	switch p.cfg.AuthFlow {
	case "device_code":
		return p.tokenViaDeviceCode(ctx)
	case "password":
		return p.tokenViaPassword(ctx)
	case "client_credentials":
		return p.tokenViaClientCredentials(ctx)
	default:
		return "", fmt.Errorf("unsupported auth_flow: %q", p.cfg.AuthFlow)
	}
}

// scopes returns the Dynamics 365 delegated permission scope.
func (p *Provider) scopes() []string {
	base := strings.TrimRight(p.cfg.CRMBaseURL, "/")
	return []string{base + "/user_impersonation"}
}

// authority returns the Azure AD authority URL for the configured tenant.
func (p *Provider) authority() string {
	return "https://login.microsoftonline.com/" + p.cfg.TenantID
}

// tokenViaDeviceCode acquires a token using the device code flow.
// Uses a file cache so the user only authenticates once (refresh token lasts ~90 days).
func (p *Provider) tokenViaDeviceCode(ctx context.Context) (string, error) {
	app, err := public.New(p.cfg.ClientID,
		public.WithAuthority(p.authority()),
		public.WithCache(p.cache),
	)
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create public client: %w", err)
	}

	scopes := p.scopes()

	// Try silent acquisition first (uses cached refresh token)
	accounts, err := app.Accounts(ctx)
	if err == nil && len(accounts) > 0 {
		result, err := app.AcquireTokenSilent(ctx, scopes,
			public.WithSilentAccount(accounts[0]))
		if err == nil {
			return result.AccessToken, nil
		}
		// Silent failed — fall through to interactive
	}

	// Interactive device code flow
	dc, err := app.AcquireTokenByDeviceCode(ctx, scopes)
	if err != nil {
		return "", fmt.Errorf("MSAL: device code initiation failed: %w", err)
	}

	// Print the user message (e.g. "Go to https://... and enter code XXXXX")
	fmt.Fprintf(os.Stderr, "\n%s\n\n", dc.Result.Message)

	result, err := dc.AuthenticationResult(ctx)
	if err != nil {
		return "", fmt.Errorf("MSAL: device code authentication failed: %w", err)
	}

	return result.AccessToken, nil
}

// tokenViaPassword acquires a token using username/password (ROPC).
// Does not work when MFA is enforced on the account.
func (p *Provider) tokenViaPassword(ctx context.Context) (string, error) {
	app, err := public.New(p.cfg.ClientID,
		public.WithAuthority(p.authority()),
		public.WithCache(p.cache),
	)
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create public client: %w", err)
	}

	scopes := p.scopes()

	// Try silent acquisition first
	accounts, err := app.Accounts(ctx)
	if err == nil {
		for _, account := range accounts {
			if strings.EqualFold(account.PreferredUsername, p.cfg.Username) {
				result, err := app.AcquireTokenSilent(ctx, scopes,
					public.WithSilentAccount(account))
				if err == nil {
					return result.AccessToken, nil
				}
				break
			}
		}
	}

	result, err := app.AcquireTokenByUsernamePassword(ctx, scopes,
		p.cfg.Username, p.cfg.Password)
	if err != nil {
		return "", fmt.Errorf("MSAL: password authentication failed: %w\n"+
			"  Hint: if MFA is enforced on your account, use auth_flow: device_code", err)
	}

	return result.AccessToken, nil
}

// tokenViaClientCredentials acquires a token using the client credentials flow.
// Requires an Azure AD app registration with a client secret.
func (p *Provider) tokenViaClientCredentials(ctx context.Context) (string, error) {
	cred, err := confidential.NewCredFromSecret(p.cfg.ClientSecret)
	if err != nil {
		return "", fmt.Errorf("MSAL: invalid client secret: %w", err)
	}

	app, err := confidential.New(p.authority(), p.cfg.ClientID, cred,
		confidential.WithCache(p.cache),
	)
	if err != nil {
		return "", fmt.Errorf("MSAL: failed to create confidential client: %w", err)
	}

	// Dynamics 365 client credentials scope uses /.default
	base := strings.TrimRight(p.cfg.CRMBaseURL, "/")
	scopes := []string{base + "/.default"}

	result, err := app.AcquireTokenSilent(ctx, scopes)
	if err != nil {
		// Silent failed — acquire fresh token
		result, err = app.AcquireTokenByCredential(ctx, scopes)
		if err != nil {
			return "", fmt.Errorf("MSAL: client credentials token acquisition failed: %w", err)
		}
	}

	return result.AccessToken, nil
}
