package grpcerror

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/piresc/gokit/apperror"
)

func MapError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.HTTPStatus {
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case 402:
			return status.Error(codes.Aborted, appErr.Message)
		case 403:
			return status.Error(codes.PermissionDenied, appErr.Message)
		case 404:
			return status.Error(codes.NotFound, appErr.Message)
		case 409:
			return status.Error(codes.AlreadyExists, appErr.Message)
		case 422:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case 503:
			return status.Error(codes.Unavailable, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}

	return status.Error(codes.Internal, err.Error())
}
