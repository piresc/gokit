package httperror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/piresc/gokit/apperror"
)

type errorResponse struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

func Write(c *gin.Context, err error) {
	WriteWithRequestID(c, err, "")
}

func WriteWithRequestID(c *gin.Context, err error, requestID string) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, errorResponse{
			Status:    "error",
			Error:     appErr.Message,
			RequestID: requestID,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, errorResponse{
		Status:    "error",
		Error:     err.Error(),
		RequestID: requestID,
	})
}
