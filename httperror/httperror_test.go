package httperror_test

import (
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
	want := `{"success":false,"error":{"code":"ERR_NOT_FOUND","message":"user not found"}}` + "\n"
	if w.Body.String() != want {
		t.Errorf("want %q, got %q", want, w.Body.String())
	}
}

func TestWrite_NonAppErrorHidesCause(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, errors.New(`pq: password authentication failed for user "admin"`))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
	want := `{"success":false,"error":{"code":"ERR_INTERNAL","message":"Internal server error"}}` + "\n"
	if w.Body.String() != want {
		t.Errorf("want %q, got %q", want, w.Body.String())
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
	want := `{"success":false,"error":{"code":"ERR_NOT_FOUND","message":"user not found"}}` + "\n"
	if w.Body.String() != want {
		t.Errorf("want %q, got %q", want, w.Body.String())
	}
}

func TestWrite_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.Write(w, apperror.ErrInternal)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}

func TestWriteWithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	httperror.WriteWithRequestID(w, apperror.Conflict("exists"), "req-9")

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", w.Code)
	}
	want := `{"success":false,"error":{"code":"ERR_CONFLICT","message":"exists"},"request_id":"req-9"}` + "\n"
	if w.Body.String() != want {
		t.Errorf("want %q, got %q", want, w.Body.String())
	}
}
