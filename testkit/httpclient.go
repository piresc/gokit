package testkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP test client with helpers for Do, AssertStatus, and ParseJSON.
type Client struct {
	baseURL string
	client  *http.Client
	token   string
}

// NewHTTPClient returns an *Client configured for a given baseURL.
// If token is set, it injects Authorization: Bearer <token> on every request.
func NewHTTPClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		token: "",
	}
}

// WithToken sets the bearer token for authenticated requests.
func (c *Client) WithToken(token string) *Client {
	c.token = token
	return c
}

// Do executes an HTTP request and returns the response.
// Injects Authorization header if token is set.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.client.Do(req)
}

// AssertStatus executes a request and asserts the response status code.
// Returns error if status doesn't match; caller must handle response body.
func (c *Client) AssertStatus(req *http.Request, wantStatus int) (*http.Response, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("expected status %d, got %d; body: %s", wantStatus, resp.StatusCode, string(body))
	}
	return resp, nil
}

// ParseJSON reads the response body and unmarshals JSON into v.
// Closes response body after reading.
func (c *Client) ParseJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("unmarshal JSON: %w", err)
	}
	return nil
}

// NewRequest creates an HTTP request with the given method and path relative to baseURL.
func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return req, nil
}

// WaitForHealth polls the given URL until it returns HTTP 200 or the timeout is exceeded.
// Polls every 1 second. Returns nil on success, or an error if timeout is reached.
func WaitForHealth(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, http.NoBody)
		if err != nil {
			return fmt.Errorf("create health check request: %w", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("health check timed out waiting for %s to return HTTP 200 after %s", url, timeout)
}
