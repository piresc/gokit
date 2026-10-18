package httperror

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/piresc/gokit/apperror"
)

const msgInternal = "internal server error"

type errorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

func Write(w http.ResponseWriter, err error) { WriteWithRequestID(w, err, "") }

func WriteWithRequestID(w http.ResponseWriter, err error, requestID string) {
	w.Header().Set("Content-Type", "application/json")

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		w.WriteHeader(appErr.HTTPStatus)
		_ = json.NewEncoder(w).Encode(errorResponse{Status: "error", Error: appErr.Message, RequestID: requestID})
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(errorResponse{Status: "error", Error: msgInternal, RequestID: requestID})
}
