package apperror

import (
	"errors"
	"net/http"

	"github.com/piresc/gokit/response"
)

var (
	ErrNotFound        = errors.New("resource not found")
	ErrAlreadyExists   = errors.New("resource already exists")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrForbidden       = errors.New("forbidden")
	ErrBadRequest      = errors.New("bad request")
	ErrInternal        = errors.New("internal server error")
	ErrTimeout         = errors.New("operation timed out")
	ErrConflict        = errors.New("resource conflict")
	ErrValidation      = errors.New("validation failed")
	ErrServiceUnavail  = errors.New("service unavailable")
	ErrRateLimited     = errors.New("rate limit exceeded")
	ErrDatabaseError   = errors.New("database error")
	ErrCacheError      = errors.New("cache error")
	ErrMessagingError  = errors.New("messaging error")
	ErrExternalService = errors.New("external service error")
	ErrInvalidToken    = errors.New("invalid token")
	ErrTokenExpired    = errors.New("token expired")
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

func New(code, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus}
}

func Internal(message string) *AppError {
	return New(response.CodeInternal, message, http.StatusInternalServerError)
}

func BadRequest(message string) *AppError {
	return New(response.CodeInvalidInput, message, http.StatusBadRequest)
}

func NotFound(message string) *AppError {
	return New(response.CodeNotFound, message, http.StatusNotFound)
}

func Forbidden(message string) *AppError {
	return New(response.CodeForbidden, message, http.StatusForbidden)
}

func Conflict(message string) *AppError {
	return New(response.CodeConflict, message, http.StatusConflict)
}

func ServiceUnavailable(message string) *AppError {
	return New(response.CodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

func PaymentFailed(message string) *AppError {
	return New(response.CodePaymentFailed, message, http.StatusPaymentRequired)
}

func UnprocessableEntity(message string) *AppError {
	return New(response.CodeUnprocessable, message, http.StatusUnprocessableEntity)
}
