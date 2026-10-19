package httperror

import (
	"errors"
	"net/http"

	"github.com/piresc/gokit/apperror"
	"github.com/piresc/gokit/response"
)

const msgInternal = "Internal server error"

func Write(w http.ResponseWriter, err error) { WriteWithRequestID(w, err, "") }

func WriteWithRequestID(w http.ResponseWriter, err error, requestID string) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		response.ErrorWithRequestID(w, appErr.HTTPStatus, appErr.Code, appErr.Message, requestID)
		return
	}

	response.ErrorWithRequestID(w, http.StatusInternalServerError, response.CodeInternal, msgInternal, requestID)
}
