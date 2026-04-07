package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shuyahonda/crm-cli/internal/auth"
	"github.com/shuyahonda/crm-cli/internal/config"
)

// Client is the Dynamics 365 Web API client.
type Client struct {
	cfg      *config.Config
	auth     *auth.Provider
	http     *http.Client
}

// ODataResponse is a generic OData collection response.
type ODataResponse[T any] struct {
	Value         []T    `json:"value"`
	NextLink      string `json:"@odata.nextLink,omitempty"`
	Count         int    `json:"@odata.count,omitempty"`
}

// NewClient creates a new Dynamics 365 Web API client.
func NewClient(cfg *config.Config, authProvider *auth.Provider) *Client {
	return &Client{
		cfg:  cfg,
		auth: authProvider,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// get performs an authenticated GET request and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, result any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// post performs an authenticated POST request.
func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

// patch performs an authenticated PATCH request.
func (c *Client) patch(ctx context.Context, path string, body any) error {
	req, err := c.newRequest(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	req.Header.Set("If-Match", "*")
	return c.do(req, nil)
}

// delete performs an authenticated DELETE request.
func (c *Client) delete(ctx context.Context, path string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// newRequest builds an authenticated HTTP request.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	token, err := c.auth.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	url := c.cfg.WebAPIBaseURL() + "/" + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("OData-MaxVersion", "4.0")
	req.Header.Set("OData-Version", "4.0")
	req.Header.Set("Prefer", "odata.include-annotations=*")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// do executes an HTTP request and decodes the response.
func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(respBody, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("CRM API error %s: %s", apiErr.Error.Code, apiErr.Error.Message)
		}
		return fmt.Errorf("CRM API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Decode response if a result container was provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
