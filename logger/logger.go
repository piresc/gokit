package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/piresc/gokit/config"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

// otelHandler wraps a base slog.Handler and injects trace_id + span_id from
// the context whenever a valid OTel span is active.
type otelHandler struct {
	base slog.Handler
}

func (h *otelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *otelHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		r.AddAttrs(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return h.base.Handle(ctx, r)
}

func (h *otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &otelHandler{base: h.base.WithAttrs(attrs)}
}

func (h *otelHandler) WithGroup(name string) slog.Handler {
	return &otelHandler{base: h.base.WithGroup(name)}
}

// fanoutHandler fans log records out to multiple slog.Handler instances.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (f *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (f *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range f.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (f *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	children := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		children[i] = h.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: children}
}

func (f *fanoutHandler) WithGroup(name string) slog.Handler {
	children := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		children[i] = h.WithGroup(name)
	}
	return &fanoutHandler{handlers: children}
}

// NewLoggerWithProvider builds a *slog.Logger with OTel trace correlation.
// If lp is non-nil, logs are also forwarded to the OTel log bridge.
// NewLogger calls this with a nil provider.
func NewLoggerWithProvider(cfg config.LoggerConfig, lp *sdklog.LoggerProvider) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	var base slog.Handler
	if cfg.Format == "json" {
		base = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		base = slog.NewTextHandler(os.Stdout, opts)
	}

	// Wrap with OTel trace correlation.
	correlated := &otelHandler{base: base}

	if lp == nil {
		return slog.New(correlated)
	}

	// Fan out to both the correlated stdout handler and the OTel log bridge.
	bridge := otelslog.NewHandler("gokit", otelslog.WithLoggerProvider(lp))
	fanout := &fanoutHandler{handlers: []slog.Handler{correlated, bridge}}
	return slog.New(fanout)
}

// NewLogger returns a *slog.Logger configured from cfg.
// Signature is unchanged; behaviour is upgraded with OTel trace correlation.
func NewLogger(cfg config.LoggerConfig) *slog.Logger {
	return NewLoggerWithProvider(cfg, nil)
}

// Err returns an slog.Attr for an error value.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
