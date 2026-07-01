package grpcmiddleware

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

type slogInterceptorLogger struct {
	logger interface {
		DebugContext(context.Context, string, ...any)
		InfoContext(context.Context, string, ...any)
		WarnContext(context.Context, string, ...any)
		ErrorContext(context.Context, string, ...any)
	}
}

func (l *slogInterceptorLogger) Log(ctx context.Context, level logging.Level, msg string, fields ...any) {
	switch level {
	case logging.LevelDebug:
		l.logger.DebugContext(ctx, msg, fields...)
	case logging.LevelInfo:
		l.logger.InfoContext(ctx, msg, fields...)
	case logging.LevelWarn:
		l.logger.WarnContext(ctx, msg, fields...)
	case logging.LevelError:
		l.logger.ErrorContext(ctx, msg, fields...)
	default:
		l.logger.InfoContext(ctx, msg, fields...)
	}
}

// LoggingUnaryInterceptor logs each RPC call on completion with duration and
// gRPC status code via go-grpc-middleware/v2.
func (i *Interceptors) LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(
		&slogInterceptorLogger{logger: i.logger},
		logging.WithLogOnEvents(logging.FinishCall),
		logging.WithFieldsFromContext(func(_ context.Context) logging.Fields { return nil }),
		logging.WithDurationField(func(duration time.Duration) logging.Fields {
			return logging.Fields{"duration_ms", duration.Milliseconds()}
		}),
	)
}
