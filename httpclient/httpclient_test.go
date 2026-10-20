package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_DefaultTimeout(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://example.com"})
	assert.NotNil(t, c)
	assert.Equal(t, 30*time.Second, c.cfg.Timeout)
}

func TestDo_GET_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	resp, err := c.Get(context.Background(), "/test")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestDo_NoRetryOn4xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	resp, _ := c.Get(context.Background(), "/test")
	assert.Equal(t, 1, calls)
	assert.Equal(t, 400, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestDo_RetryOn5xx(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL, Timeout: 5 * time.Second})
	resp, _ := c.Get(context.Background(), "/test")
	assert.Equal(t, 3, calls)
	_ = resp.Body.Close()
}

func TestGetJSON_EnvelopeUnwrap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"abc"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct {
		ID string `json:"id"`
	}
	err := c.GetJSON(context.Background(), "/test", &result)
	assert.NoError(t, err)
	assert.Equal(t, "abc", result.ID)
}

func TestGetJSON_DirectUnmarshal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"xyz"}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct {
		ID string `json:"id"`
	}
	err := c.GetJSON(context.Background(), "/test", &result)
	assert.NoError(t, err)
	assert.Equal(t, "xyz", result.ID)
}

func TestDo_APIKeyHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my-key", r.Header.Get("X-API-Key"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL, APIKey: "my-key"})
	resp, err := c.Get(context.Background(), "/test")
	assert.NoError(t, err)
	_ = resp.Body.Close()
}

func TestDo_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "/test")
	assert.Error(t, err)
}

func TestGetJSON_EnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"ERR_NOT_FOUND","message":"not found"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	result := map[string]any{}
	err := c.GetJSON(context.Background(), "/test", &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "ERR_NOT_FOUND", apiErr.Code)
	assert.Equal(t, "not found", apiErr.Message)
	assert.Equal(t, 200, apiErr.StatusCode)
	assert.Empty(t, result, "success:false body must never be decoded into result")
}

func TestGetJSON_EnvelopeErrorNilResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"ERR_X","message":"boom"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	err := c.GetJSON(context.Background(), "/test", nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "ERR_X", apiErr.Code)
}

func TestGetJSON_EnvelopeErrorStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"ERR_NOT_FOUND","message":"msg"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	err := c.GetJSON(context.Background(), "/test", nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Equal(t, "API error 404: ERR_NOT_FOUND: msg", err.Error())
}

func TestGetJSON_EnvelopeErrorFallsBackToStatusText(t *testing.T) {
	for name, body := range map[string]string{
		"missing": `{"success":false}`,
		"empty":   `{"success":false,"error":{}}`,
		"invalid": `{"success":false,"error":123}`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			c := NewClient(Config{BaseURL: srv.URL})
			err := c.GetJSON(context.Background(), "/test", nil)
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, "Bad Request", apiErr.Message)
			assert.Equal(t, "API error 400: Bad Request", err.Error())
		})
	}
}

func TestGetJSON_StatusErrorBodyReadFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	err := c.GetJSON(context.Background(), "/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error 400")
}

func TestGetJSON_EnvelopeErrorOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"ERR_NOT_FOUND","message":"user not found"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct{}
	err := c.GetJSON(context.Background(), "/test", &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "ERR_NOT_FOUND", apiErr.Code)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetJSON_EnvelopeSuccessOmittedData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result []string
	assert.NoError(t, c.GetJSON(context.Background(), "/test", &result))
	assert.Empty(t, result)
}

func TestGetJSON_HTTPErrorPlainBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`plain text`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct{}
	err := c.GetJSON(context.Background(), "/test", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error 400")
}

func TestDo_ContentTypeAndAcceptHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	resp, err := c.Get(context.Background(), "/test")
	assert.NoError(t, err)
	_ = resp.Body.Close()
}

func TestPostJSON_EnvelopeUnwrap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"token":"abc123"}}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct {
		Token string `json:"token"`
	}
	err := c.PostJSON(context.Background(), "/login", map[string]string{"user": "test"}, &result)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", result.Token)
}

func TestPutJSON_DirectUnmarshal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	var result struct {
		Updated bool `json:"updated"`
	}
	err := c.PutJSON(context.Background(), "/resource/1", map[string]string{"name": "new"}, &result)
	assert.NoError(t, err)
	assert.True(t, result.Updated)
}

func TestDo_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	resp, err := c.Delete(context.Background(), "/resource/1")
	assert.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	_ = resp.Body.Close()
}

func TestAPIKeyCustomHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("x-cg-demo-api-key"))
		assert.Empty(t, r.Header.Get("X-API-Key"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL, APIKey: "k", APIKeyHeader: "x-cg-demo-api-key"})
	resp, err := c.Get(context.Background(), "/test")
	require.NoError(t, err)
	_ = resp.Body.Close()
}

func TestAPIKeyDefaultHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "k", r.Header.Get("X-API-Key"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL, APIKey: "k"})
	resp, err := c.Get(context.Background(), "/test")
	require.NoError(t, err)
	_ = resp.Body.Close()
}
