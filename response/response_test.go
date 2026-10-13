package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piresc/gokit/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	response.Success(w, http.StatusOK, map[string]string{"id": "1"})

	assert.Equal(t, http.StatusOK, w.Code)
	var body response.SuccessResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "success", body.Status)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	response.Error(w, http.StatusBadRequest, "bad input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body response.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "bad input", body.Error)
}

func TestErrorWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	response.ErrorWithRequestID(w, http.StatusInternalServerError, "oops", "req-123")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body response.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "req-123", body.RequestID)
}
