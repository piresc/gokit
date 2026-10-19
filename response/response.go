// Package response writes the JSON envelope shared by all services.
//
// Success: {"success":true,"data":...}
// Error:   {"success":false,"error":{"code":"ERR_...","message":"..."}}
package response

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// Error codes. Services may define additional ERR_* codes of their own.
const (
	CodeUnauthorized       = "ERR_UNAUTHORIZED"
	CodeTokenExpired       = "ERR_TOKEN_EXPIRED"
	CodeTokenRevoked       = "ERR_TOKEN_REVOKED"
	CodeForbidden          = "ERR_FORBIDDEN"
	CodeInvalidInput       = "ERR_INVALID_INPUT"
	CodeNotFound           = "ERR_NOT_FOUND"
	CodeConflict           = "ERR_CONFLICT"
	CodeRateLimited        = "ERR_RATE_LIMITED"
	CodeInternal           = "ERR_INTERNAL"
	CodeServiceUnavailable = "ERR_SERVICE_UNAVAILABLE"
	CodePaymentFailed      = "ERR_PAYMENT_FAILED"
	CodeUnprocessable      = "ERR_UNPROCESSABLE"
)

// SuccessResponse is the success envelope. Data is omitted when empty.
type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
}

// ErrorBody is the error detail inside ErrorResponse.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the error envelope.
type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
}

// Success writes the success envelope. Data that encoding/json would treat as
// empty under omitempty (nil, empty slice/map/string, false, 0, nil pointer)
// is left out, matching a generic Data T with omitempty. Zero structs are kept.
func Success(w http.ResponseWriter, statusCode int, data any) {
	if isEmpty(data) {
		data = nil
	}
	write(w, statusCode, SuccessResponse{Success: true, Data: data})
}

// Error writes the error envelope.
func Error(w http.ResponseWriter, statusCode int, code, message string) {
	write(w, statusCode, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

// ErrorWithRequestID writes the error envelope with a top-level request_id.
func ErrorWithRequestID(w http.ResponseWriter, statusCode int, code, message, requestID string) {
	write(w, statusCode, ErrorResponse{Error: ErrorBody{Code: code, Message: message}, RequestID: requestID})
}

func write(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

// isEmpty reports whether encoding/json's omitempty would drop v.
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array, reflect.String:
		return rv.Len() == 0
	case reflect.Pointer:
		return rv.IsNil()
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	}
	return false
}
