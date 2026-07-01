package grpcmiddleware_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/piresc/gokit/grpcmiddleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ---- helpers ----

func newInterceptors() *grpcmiddleware.Interceptors {
	return grpcmiddleware.NewInterceptors("", nil, nil, nil)
}

func unaryInfo(method string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: method}
}

func okHandler(_ context.Context, _ interface{}) (interface{}, error) {
	return "ok", nil
}

// ---- recovery ----

func TestRecovery_PanicString(t *testing.T) {
	i := newInterceptors()
	resp, err := i.RecoveryUnaryInterceptor()(context.Background(), nil, unaryInfo("/s/m"),
		func(_ context.Context, _ interface{}) (interface{}, error) { panic("boom") })
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRecovery_PanicError(t *testing.T) {
	i := newInterceptors()
	resp, err := i.RecoveryUnaryInterceptor()(context.Background(), nil, unaryInfo("/s/m"),
		func(_ context.Context, _ interface{}) (interface{}, error) { panic(errors.New("err panic")) })
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestRecovery_NoPanic(t *testing.T) {
	i := newInterceptors()
	resp, err := i.RecoveryUnaryInterceptor()(context.Background(), nil, unaryInfo("/s/m"), okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// ---- auth ----

func bearerCtx(token string) context.Context {
	md := metadata.Pairs("authorization", "Bearer "+token)
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestAuth_PublicMethod_Passthrough(t *testing.T) {
	i := grpcmiddleware.NewInterceptors("secret", nil, nil, nil)
	public := map[string]bool{"/svc.S/Login": true}
	resp, err := i.AuthUnaryInterceptor(public)(context.Background(), nil, unaryInfo("/svc.S/Login"), okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestAuth_MissingToken(t *testing.T) {
	i := grpcmiddleware.NewInterceptors("secret", nil, nil, nil)
	_, err := i.AuthUnaryInterceptor(nil)(context.Background(), nil, unaryInfo("/svc.S/Protected"), okHandler)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuth_InvalidToken(t *testing.T) {
	i := grpcmiddleware.NewInterceptors("secret", nil, nil, nil)
	ctx := bearerCtx("not-a-jwt")
	_, err := i.AuthUnaryInterceptor(nil)(ctx, nil, unaryInfo("/svc.S/Protected"), okHandler)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

// ---- context helpers ----

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = grpcmiddleware.WithUserID(ctx, "u42")
	ctx = grpcmiddleware.WithRole(ctx, "admin")
	assert.Equal(t, "u42", grpcmiddleware.UserIDFromContext(ctx))
	assert.Equal(t, "admin", grpcmiddleware.RoleFromContext(ctx))
}

// ---- rate limit ----

func TestRateLimit_Allow(t *testing.T) {
	i := newInterceptors()
	interceptor := i.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
	})
	resp, err := interceptor(context.Background(), nil, unaryInfo("/s/m"), okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestRateLimit_Reject(t *testing.T) {
	i := newInterceptors()
	interceptor := i.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
	})
	// first request consumes the burst
	_, _ = interceptor(context.Background(), nil, unaryInfo("/s/m"), okHandler)
	// second should be rejected
	_, err := interceptor(context.Background(), nil, unaryInfo("/s/m"), okHandler)
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
}

// ---- logging (smoke) ----

func TestLogging_NoError(t *testing.T) {
	i := grpcmiddleware.NewInterceptors("", slog.Default(), nil, nil)
	resp, err := i.LoggingUnaryInterceptor()(context.Background(), nil, unaryInfo("/s/m"), okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// ---- tracing (smoke) ----

func TestTracing_NoError(t *testing.T) {
	i := newInterceptors()
	resp, err := i.TracingUnaryInterceptor()(context.Background(), nil, unaryInfo("/pkg.Svc/Method"), okHandler)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

// ---- shutdown is safe ----

func TestShutdown_Safe(t *testing.T) {
	i := newInterceptors()
	assert.NotPanics(t, i.Shutdown)
}

// ---- concurrent safety ----

func TestRateLimit_Concurrent(t *testing.T) {
	i := newInterceptors()
	interceptor := i.RateLimitUnaryInterceptor(grpcmiddleware.RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         1000,
	})
	var wg sync.WaitGroup
	for j := 0; j < 20; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = interceptor(context.Background(), nil, unaryInfo("/s/m"), okHandler)
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent test timed out")
	}
}
