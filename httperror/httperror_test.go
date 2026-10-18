package httperror_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/piresc/gokit/apperror"
	"github.com/piresc/gokit/httperror"
)

func TestWrite_AppError(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, apperror.NotFound("user not found"))

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "error" {
		t.Errorf("want status=error, got %s", body["status"])
	}
	if body["error"] != "user not found" {
		t.Errorf("want 'user not found', got %s", body["error"])
	}
}

func TestWrite_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, apperror.ErrInternal)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}
