package grpcerror_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/piresc/gokit/apperror"
	"github.com/piresc/gokit/grpcerror"
)

func TestMapError_Nil(t *testing.T) {
	if err := grpcerror.MapError(nil); err != nil {
		t.Errorf("want nil, got %v", err)
	}
}

func TestMapError_AppErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"BadRequest", apperror.BadRequest("bad"), codes.InvalidArgument},
		{"NotFound", apperror.NotFound("missing"), codes.NotFound},
		{"Forbidden", apperror.Forbidden("denied"), codes.PermissionDenied},
		{"Conflict", apperror.Conflict("exists"), codes.AlreadyExists},
		{"ServiceUnavailable", apperror.ServiceUnavailable("down"), codes.Unavailable},
		{"PaymentFailed", apperror.PaymentFailed("declined"), codes.Aborted},
		{"Internal", apperror.Internal("oops"), codes.Internal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcErr := grpcerror.MapError(tt.err)
			st, _ := status.FromError(grpcErr)
			if st.Code() != tt.wantCode {
				t.Errorf("want %v, got %v", tt.wantCode, st.Code())
			}
		})
	}
}

func TestMapError_GenericError(t *testing.T) {
	grpcErr := grpcerror.MapError(apperror.ErrInternal)
	st, _ := status.FromError(grpcErr)
	if st.Code() != codes.Internal {
		t.Errorf("want Internal, got %v", st.Code())
	}
}
