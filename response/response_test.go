package response_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piresc/gokit/response"
	"github.com/stretchr/testify/assert"
)

type item struct {
	ID string `json:"id"`
}

func TestSuccess_Golden(t *testing.T) {
	tests := []struct {
		name string
		data any
		want string
	}{
		{"map", map[string]string{"id": "1"}, `{"success":true,"data":{"id":"1"}}` + "\n"},
		{"nil", nil, `{"success":true}` + "\n"},
		{"empty slice", []item{}, `{"success":true}` + "\n"},
		{"nil slice", []item(nil), `{"success":true}` + "\n"},
		{"empty map", map[string]int{}, `{"success":true}` + "\n"},
		{"nil pointer", (*item)(nil), `{"success":true}` + "\n"},
		{"non-empty slice", []item{{ID: "1"}}, `{"success":true,"data":[{"id":"1"}]}` + "\n"},
		{"zero struct", item{}, `{"success":true,"data":{"id":""}}` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			response.Success(w, http.StatusOK, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			assert.Equal(t, tt.want, w.Body.String())
		})
	}
}

func TestError_Golden(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, http.StatusBadRequest, response.CodeInvalidInput, "bad input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":false,"error":{"code":"ERR_INVALID_INPUT","message":"bad input"}}`+"\n", w.Body.String())
}

func TestErrorWithRequestID_Golden(t *testing.T) {
	w := httptest.NewRecorder()
	response.ErrorWithRequestID(w, http.StatusInternalServerError, response.CodeInternal, "oops", "req-123")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":false,"error":{"code":"ERR_INTERNAL","message":"oops"},"request_id":"req-123"}`+"\n", w.Body.String())
}

func TestCodes(t *testing.T) {
	assert.Equal(t, "ERR_UNAUTHORIZED", response.CodeUnauthorized)
	assert.Equal(t, "ERR_TOKEN_EXPIRED", response.CodeTokenExpired)
	assert.Equal(t, "ERR_TOKEN_REVOKED", response.CodeTokenRevoked)
	assert.Equal(t, "ERR_FORBIDDEN", response.CodeForbidden)
	assert.Equal(t, "ERR_INVALID_INPUT", response.CodeInvalidInput)
	assert.Equal(t, "ERR_NOT_FOUND", response.CodeNotFound)
	assert.Equal(t, "ERR_CONFLICT", response.CodeConflict)
	assert.Equal(t, "ERR_RATE_LIMITED", response.CodeRateLimited)
	assert.Equal(t, "ERR_INTERNAL", response.CodeInternal)
	assert.Equal(t, "ERR_SERVICE_UNAVAILABLE", response.CodeServiceUnavailable)
	assert.Equal(t, "ERR_PAYMENT_FAILED", response.CodePaymentFailed)
	assert.Equal(t, "ERR_UNPROCESSABLE", response.CodeUnprocessable)
}
