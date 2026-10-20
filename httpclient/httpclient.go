package httpclient

import (
	"bytes"
	"cmp"
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
	BaseURL      string
	APIKey       string
	APIKeyHeader string // header carrying APIKey; defaults to "X-API-Key"
	Timeout      time.Duration
	Tracer       tracing.Tracer // optional, may be nil
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
		req.Header.Set(cmp.Or(c.cfg.APIKeyHeader, "X-API-Key"), c.cfg.APIKey)
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
		return statusError(resp)
	}

	return decodeJSON(resp, result)
}

// PostJSON performs a POST request with a JSON body and decodes the JSON response into result.
func (c *Client) PostJSON(ctx context.Context, endpoint string, body, result any) error {
	resp, err := c.Post(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return statusError(resp)
	}

	return decodeJSON(resp, result)
}

// PutJSON performs a PUT request with a JSON body and decodes the JSON response into result.
func (c *Client) PutJSON(ctx context.Context, endpoint string, body, result any) error {
	resp, err := c.Put(ctx, endpoint, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return statusError(resp)
	}

	return decodeJSON(resp, result)
}

// APIError is returned when a response carries the error envelope
// {"success":false,"error":{"code":"ERR_...","message":"..."}}.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error %d: %s: %s", e.StatusCode, e.Code, e.Message)
}

// statusText returns the HTTP status text without the numeric prefix.
func statusText(resp *http.Response) string {
	if text := http.StatusText(resp.StatusCode); text != "" {
		return text
	}
	return strings.TrimSpace(strings.TrimPrefix(resp.Status, fmt.Sprint(resp.StatusCode)))
}

// envelopeError extracts the APIError from a success:false envelope body.
// It returns nil when the body is not an error envelope. A missing or
// unparseable error object falls back to the HTTP status text.
func envelopeError(body []byte, resp *http.Response) *APIError {
	var raw map[string]json.RawMessage
	if json.Unmarshal(body, &raw) != nil {
		return nil
	}
	if _, ok := raw["success"]; !ok {
		return nil
	}
	var envelope struct {
		Success bool            `json:"success"`
		Error   json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &envelope) != nil || envelope.Success {
		return nil
	}
	apiErr := &APIError{StatusCode: resp.StatusCode}
	var detail struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(envelope.Error, &detail) == nil {
		apiErr.Code, apiErr.Message = detail.Code, detail.Message
	} else {
		// Tolerate a bare string error from older servers.
		_ = json.Unmarshal(envelope.Error, &apiErr.Message)
	}
	if apiErr.Code == "" && apiErr.Message == "" {
		apiErr.Message = statusText(resp)
	}
	return apiErr
}

// statusError builds the error for a 4xx/5xx response, preferring the
// envelope's code and message over the bare HTTP status.
func statusError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		if apiErr := envelopeError(body, resp); apiErr != nil {
			return apiErr
		}
	}
	return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
}

// decodeJSON tries to decode the response as an API envelope first.
// Envelope: {"success": bool, "data": ..., "error": {"code": "...", "message": "..."}}.
// If the body has a "success" field and success is false, an *APIError is returned
// and the raw body is never decoded into result (even when result is nil).
// If the body has no "success" field, a direct unmarshal into result is attempted.
func decodeJSON(resp *http.Response, result any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if apiErr := envelopeError(body, resp); apiErr != nil {
		return apiErr
	}

	if result == nil {
		return nil
	}

	// Peek at the top-level keys to detect a success envelope without relying
	// on the zero-value of bool.
	var raw map[string]json.RawMessage
	if json.Unmarshal(body, &raw) == nil {
		if _, hasSuccess := raw["success"]; hasSuccess {
			var envelope struct {
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(body, &envelope); err == nil {
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
