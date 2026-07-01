package grpcclient

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/piresc/gokit/tracing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ClientConfig holds all parameters for Dial.
type ClientConfig struct {
	Target           string
	DialTimeout      time.Duration
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
	TLSEnabled       bool
	Tracer           tracing.Tracer
	Logger           *slog.Logger
}

// Dial creates a gRPC client connection with OTel tracing, logging, and auth
// forwarding interceptors. Extra dial options are appended after the defaults.
func Dial(ctx context.Context, cfg ClientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			clientAuthForwardingInterceptor(),
			clientLoggingUnaryInterceptor(logger),
		),
		grpc.WithChainStreamInterceptor(
			clientAuthForwardingStreamInterceptor(),
			clientLoggingStreamInterceptor(logger),
		),
	}

	if !cfg.TLSEnabled {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.KeepAliveTime > 0 || cfg.KeepAliveTimeout > 0 {
		dialOpts = append(dialOpts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    cfg.KeepAliveTime,
			Timeout: cfg.KeepAliveTimeout,
		}))
	}

	dialOpts = append(dialOpts, opts...)

	if cfg.DialTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.DialTimeout)
		defer cancel()
	}

	conn, err := grpc.NewClient(cfg.Target, dialOpts...)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("dial timeout: %s", cfg.Target)
		}
		return nil, fmt.Errorf("failed to dial %s: %w", cfg.Target, err)
	}

	return conn, nil
}

func clientLoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", method),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			logger.LogAttrs(ctx, slog.LevelInfo, "grpc client call completed", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelError, "grpc client call completed", attrs...)
		}

		return err
	}
}

func clientLoggingStreamInterceptor(logger *slog.Logger) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()
		cs, err := streamer(ctx, desc, cc, method, opts...)
		duration := time.Since(start)

		st, _ := status.FromError(err)
		code := codes.OK
		if st != nil {
			code = st.Code()
		}

		attrs := []slog.Attr{
			slog.String("grpc.method", method),
			slog.String("grpc.code", code.String()),
			slog.Float64("duration_ms", float64(duration.Milliseconds())),
		}

		if code == codes.OK {
			logger.LogAttrs(ctx, slog.LevelInfo, "grpc client stream started", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelError, "grpc client stream failed", attrs...)
		}

		return cs, err
	}
}

// clientAuthForwardingInterceptor propagates an incoming "authorization"
// metadata value to the outgoing call (gateway → service forwarding).
func clientAuthForwardingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("authorization"); len(vals) > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", vals[0])
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func clientAuthForwardingStreamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("authorization"); len(vals) > 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, "authorization", vals[0])
			}
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
