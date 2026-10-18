package response

import (
	"encoding/json"
	"net/http"
)

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

type ErrorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

func Success(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(SuccessResponse{Status: "success", Data: data})
}

func Error(w http.ResponseWriter, statusCode int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Status: "error", Error: err})
}

func ErrorWithRequestID(w http.ResponseWriter, statusCode int, err, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Status: "error", Error: err, RequestID: requestID})
}
