package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/piresc/gokit/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

func newContext(w *httptest.ResponseRecorder) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c := newContext(w)
	response.Success(c, http.StatusOK, gin.H{"id": "1"})

	assert.Equal(t, http.StatusOK, w.Code)
	var body response.SuccessResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "success", body.Status)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c := newContext(w)
	response.Error(c, http.StatusBadRequest, "bad input")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body response.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "error", body.Status)
	assert.Equal(t, "bad input", body.Error)
}

func TestErrorWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c := newContext(w)
	response.ErrorWithRequestID(c, http.StatusInternalServerError, "oops", "req-123")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body response.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "req-123", body.RequestID)
}
