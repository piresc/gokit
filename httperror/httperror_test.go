package httperror_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestWrite_NonAppErrorHidesCause(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, errors.New(`pq: password authentication failed for user "admin"`))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "error" {
		t.Errorf("want status=error, got %s", body["status"])
	}
	if body["error"] != "internal server error" {
		t.Errorf("want 'internal server error', got %s", body["error"])
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Errorf("body leaks cause: %s", w.Body.String())
	}
}

func TestWrite_WrappedAppErrorKeepsMessage(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, fmt.Errorf("load: %w", apperror.NotFound("user not found")))

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
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
