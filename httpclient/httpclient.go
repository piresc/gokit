package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/piresc/gokit/tracing"
)

// Config holds configuration for the HTTP client.
type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
	Tracer  tracing.Tracer // optional, may be nil
}

// Client is an HTTP client with retry logic and optional distributed tracing.
type Client struct {
	httpClient *http.Client
	cfg        Config
}

// NewClient creates a new Client. If cfg.Timeout is zero, it defaults to 30s.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		cfg:        cfg,
	}
}

// Do executes an HTTP request with up to 3 attempts and exponential backoff on 5xx.
// Body is marshaled to JSON when non-nil. 4xx responses are returned immediately without retry.
func (c *Client) Do(ctx context.Context, method, endpoint string, body any) (*http.Response, error) {
	url := c.cfg.BaseURL + endpoint

	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}

	if c.cfg.Tracer != nil && c.cfg.Tracer.IsEnabled() {
		c.cfg.Tracer.InjectContext(ctx, req.Header)
	}

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		// Reset the body reader for each attempt.
		if jsonBody != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(jsonBody))
			req.ContentLength = int64(len(jsonBody))
		}

		attemptCtx := ctx
		var endSegment func()
		if c.cfg.Tracer != nil && c.cfg.Tracer.IsEnabled() {
			host := extractHost(url)
			attemptCtx, endSegment = c.cfg.Tracer.StartExternalCall(ctx, host, method)
		}

		resp, err = c.httpClient.Do(req.WithContext(attemptCtx))

		if endSegment != nil {
			endSegment()
		}

		// 2xx, 3xx, and 4xx are returned immediately — only 5xx and transport errors retry.
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// On 5xx or network error, drain and close the body before retrying.
		if attempt < 2 {
			if resp != nil {
				_ = resp.Body.Close()
				resp = nil
			}

			// Backoff: 100ms after attempt 0, 200ms after attempt 1.
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return resp, err
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, endpoint, nil)
}

// Post performs a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, endpoint, body)
}

// Put performs a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, endpoint string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, endpoint, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, endpoint string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, endpoint, nil)
}

// GetJSON performs a GET request and decodes the JSON response into result.
func (c *Client) GetJSON(ctx context.Context, endpoint string, result any) error {
	resp, err := c.Get(ctx, endpoint)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return decodeJSON(resp.Body, result)
}

// PostJSON performs a POST request with a JSON body and decodes the JSON response into result.
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result any) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return decodeJSON(resp.Body, result)
}

// PutJSON performs a PUT request with a JSON body and decodes the JSON response into result.
func (c *Client) PutJSON(ctx context.Context, endpoint string, body, result any) error {
	resp, err := c.Put(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return decodeJSON(resp.Body, result)
}

// decodeJSON tries to decode the response as an API envelope first.
// Envelope: {"success": bool, "data": ..., "error": "..."}.
// If the body has a "success" field and success is false, an error is returned.
// If the body has no "success" field, a direct unmarshal into result is attempted.
func decodeJSON(r io.Reader, result any) error {
	if result == nil {
		return nil
	}

	body, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Peek at the top-level keys to detect an envelope without relying on
	// the zero-value of bool (success=false would be indistinguishable from
	// a missing field if we decoded directly into a struct).
	var raw map[string]json.RawMessage
	if json.Unmarshal(body, &raw) == nil {
		if _, hasSuccess := raw["success"]; hasSuccess {
			var envelope struct {
				Success bool            `json:"success"`
				Data    json.RawMessage `json:"data"`
				Error   string          `json:"error"`
			}
			if err := json.Unmarshal(body, &envelope); err == nil {
				if !envelope.Success {
					return fmt.Errorf("API error: %s", envelope.Error)
				}
				if len(envelope.Data) > 0 {
					if err := json.Unmarshal(envelope.Data, result); err != nil {
						return fmt.Errorf("decode JSON response data: %w", err)
					}
				}
				return nil
			}
		}
	}

	// Fall back to direct unmarshal.
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}
	return nil
}

// extractHost strips the scheme and path from a URL, returning only the host.
func extractHost(rawURL string) string {
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if i := strings.Index(rawURL, "/"); i != -1 {
		return rawURL[:i]
	}
	return rawURL
}
