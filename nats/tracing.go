package nats

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/piresc/gokit/nats"

// natsHeaderCarrier adapts nats.Header to propagation.TextMapCarrier. Keys are
// stored exactly as given: NATS headers are case-sensitive, so http.Header
// canonicalisation would turn "traceparent" into "Traceparent".
type natsHeaderCarrier nats.Header

// Get returns the value for key, or "" if absent.
func (c natsHeaderCarrier) Get(key string) string {
	if v := c[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set stores value under key exactly as given.
func (c natsHeaderCarrier) Set(key, value string) { c[key] = []string{value} }

// Keys returns all header keys.
func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// MessageContext extracts the remote span context from msg's headers and
// returns it attached to ctx.
func MessageContext(ctx context.Context, msg jetstream.Msg) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, natsHeaderCarrier(msg.Headers()))
}

func messagingAttrs(subject string) trace.SpanStartEventOption {
	return trace.WithAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.destination.name", subject),
	)
}
