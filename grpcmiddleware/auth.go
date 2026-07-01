package grpcmiddleware

import (
	"context"
	"strings"

	"github.com/piresc/gokit/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthUnaryInterceptor validates a Bearer JWT from gRPC metadata and stores
// UserID + Role into the context. publicMethods is a map of full method names
// (e.g. "/svc.Service/Login") that bypass authentication.
func (i *Interceptors) AuthUnaryInterceptor(publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		newCtx, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

func (i *Interceptors) authenticate(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Errorf(codes.Unauthenticated, "missing authorization metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return ctx, status.Errorf(codes.Unauthenticated, "missing authorization metadata")
	}

	parts := strings.SplitN(values[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ctx, status.Errorf(codes.Unauthenticated, "invalid or expired token")
	}

	claims, err := auth.ValidateToken(parts[1], i.jwtSecret)
	if err != nil {
		return ctx, status.Errorf(codes.Unauthenticated, "invalid or expired token")
	}

	newCtx := context.WithValue(ctx, CtxKeyUserID, claims.UserID)
	newCtx = context.WithValue(newCtx, CtxKeyRole, claims.Role)
	return newCtx, nil
}

// wrappedStream replaces the stream's context with a derived one carrying auth
// values — used by stream interceptors.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context { return w.ctx }
