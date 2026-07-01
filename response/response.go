package response

import "github.com/gin-gonic/gin"

// SuccessResponse is the standard JSON envelope for 2xx responses.
type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

// ErrorResponse is the standard JSON envelope for error responses.
type ErrorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// Success writes {"status":"success","data":data} with the given status code.
func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Status: "success",
		Data:   data,
	})
}

// Error writes {"status":"error","error":err} with the given status code.
func Error(c *gin.Context, statusCode int, err string) {
	c.JSON(statusCode, ErrorResponse{
		Status: "error",
		Error:  err,
	})
}

// ErrorWithRequestID writes an error response that includes a request ID.
func ErrorWithRequestID(c *gin.Context, statusCode int, err, requestID string) {
	c.JSON(statusCode, ErrorResponse{
		Status:    "error",
		Error:     err,
		RequestID: requestID,
	})
}
