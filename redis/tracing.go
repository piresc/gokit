package redis

import (
	"context"
	"errors"
	"net"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/piresc/gokit/redis"

// tracingHook creates client spans for commands issued under a context that
// already holds a valid span. Command arguments are never recorded.
type tracingHook struct{}

var _ redis.Hook = tracingHook{}

func (tracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

func (tracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !trace.SpanContextFromContext(ctx).IsValid() {
			return next(ctx, cmd)
		}
		name := cmd.Name()
		ctx, span := startSpan(ctx, name, name)
		defer span.End()
		err := next(ctx, cmd)
		recordErr(span, err)
		return err
	}
}

func (tracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !trace.SpanContextFromContext(ctx).IsValid() {
			return next(ctx, cmds)
		}
		ctx, span := startSpan(ctx, "pipeline", "pipeline")
		defer span.End()
		err := next(ctx, cmds)
		recordErr(span, err)
		return err
	}
}

func startSpan(ctx context.Context, name, operation string) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", operation),
		),
	)
}

func recordErr(span trace.Span, err error) {
	if err == nil || errors.Is(err, redis.Nil) {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
