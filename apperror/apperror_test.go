package apperror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/piresc/gokit/apperror"
)

func TestNew(t *testing.T) {
	err := apperror.New("TEST_ERR", "test message", http.StatusConflict)
	if err.Code != "TEST_ERR" {
		t.Errorf("want code TEST_ERR, got %s", err.Code)
	}
	if err.Message != "test message" {
		t.Errorf("want message 'test message', got %s", err.Message)
	}
	if err.HTTPStatus != 409 {
		t.Errorf("want status 409, got %d", err.HTTPStatus)
	}
}

func TestErrorInterface(t *testing.T) {
	err := apperror.BadRequest("invalid input")
	if err.Error() != "invalid input" {
		t.Errorf("want 'invalid input', got %s", err.Error())
	}
}

func TestErrorsAs(t *testing.T) {
	err := apperror.NotFound("user not found")
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatal("expected errors.As to match AppError")
	}
	if appErr.HTTPStatus != http.StatusNotFound {
		t.Errorf("want 404, got %d", appErr.HTTPStatus)
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) *apperror.AppError
		status int
		code   string
	}{
		{"Internal", apperror.Internal, 500, "INTERNAL_ERROR"},
		{"BadRequest", apperror.BadRequest, 400, "BAD_REQUEST"},
		{"NotFound", apperror.NotFound, 404, "NOT_FOUND"},
		{"Forbidden", apperror.Forbidden, 403, "FORBIDDEN"},
		{"Conflict", apperror.Conflict, 409, "CONFLICT"},
		{"ServiceUnavailable", apperror.ServiceUnavailable, 503, "SERVICE_UNAVAILABLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("msg")
			if err.HTTPStatus != tt.status {
				t.Errorf("want %d, got %d", tt.status, err.HTTPStatus)
			}
			if err.Code != tt.code {
				t.Errorf("want %s, got %s", tt.code, err.Code)
			}
		})
	}
}
