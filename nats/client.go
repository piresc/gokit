// Package nats provides a NATS JetStream client wrapper.
package nats

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client wraps a NATS connection with JetStream support.
type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream

	mu        sync.RWMutex
	streams   map[string]jetstream.Stream
	consumers map[string]jetstream.Consumer
}

// NewClient connects to NATS at url and initialises JetStream.
func NewClient(url string, opts ...nats.Option) (*Client, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	slog.Info("nats connected", "url", url)

	return &Client{
		conn:      nc,
		js:        js,
		streams:   make(map[string]jetstream.Stream),
		consumers: make(map[string]jetstream.Consumer),
	}, nil
}

// JetStream returns the underlying JetStream context.
func (c *Client) JetStream() jetstream.JetStream { return c.js }

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *nats.Conn { return c.conn }

// CreateStream creates or updates a JetStream stream.
func (c *Client) CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	stream, err := c.js.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create stream %s: %w", cfg.Name, err)
	}

	c.mu.Lock()
	c.streams[cfg.Name] = stream
	c.mu.Unlock()

	slog.Info("stream created", "name", cfg.Name, "subjects", cfg.Subjects)
	return stream, nil
}

// CreateConsumer creates or updates a durable consumer on streamName.
func (c *Client) CreateConsumer(ctx context.Context, streamName string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	c.mu.RLock()
	stream, ok := c.streams[streamName]
	c.mu.RUnlock()

	if !ok {
		var err error
		stream, err = c.js.Stream(ctx, streamName)
		if err != nil {
			return nil, fmt.Errorf("get stream %s: %w", streamName, err)
		}
		c.mu.Lock()
		c.streams[streamName] = stream
		c.mu.Unlock()
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer %s on stream %s: %w", cfg.Name, streamName, err)
	}

	c.mu.Lock()
	c.consumers[cfg.Name] = consumer
	c.mu.Unlock()

	slog.Info("consumer created", "name", cfg.Name, "stream", streamName)
	return consumer, nil
}

// Publish publishes data to subject, optionally deduplicating by msgID.
func (c *Client) Publish(ctx context.Context, subject string, data []byte, msgID string) (*jetstream.PubAck, error) {
	var opts []jetstream.PublishOpt
	if msgID != "" {
		opts = append(opts, jetstream.WithMsgID(msgID))
	}

	msg := nats.NewMsg(subject)
	msg.Data = data

	var span trace.Span
	if trace.SpanContextFromContext(ctx).IsValid() {
		ctx, span = otel.Tracer(tracerName).Start(ctx, subject+" publish",
			trace.WithSpanKind(trace.SpanKindProducer),
			messagingAttrs(subject))
		defer span.End()
		otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg.Header))
	}

	ack, err := c.js.PublishMsg(ctx, msg, opts...)
	if err != nil {
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return nil, fmt.Errorf("publish to %s: %w", subject, err)
	}
	return ack, nil
}

// ConsumeContext is like Consume but passes handler a context carrying the
// remote trace context from the message headers, with a consumer span active.
func (c *Client) ConsumeContext(consumerName string, handler func(ctx context.Context, msg jetstream.Msg)) (jetstream.ConsumeContext, error) {
	return c.Consume(consumerName, func(msg jetstream.Msg) {
		ctx := MessageContext(context.Background(), msg)
		ctx, span := otel.Tracer(tracerName).Start(ctx, msg.Subject()+" process",
			trace.WithSpanKind(trace.SpanKindConsumer),
			messagingAttrs(msg.Subject()))
		defer span.End()
		handler(ctx, msg)
	})
}

// Consume starts consuming messages for consumerName using handler.
func (c *Client) Consume(consumerName string, handler func(jetstream.Msg)) (jetstream.ConsumeContext, error) {
	c.mu.RLock()
	consumer, ok := c.consumers[consumerName]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("consumer %s not found, create it first", consumerName)
	}

	cc, err := consumer.Consume(handler)
	if err != nil {
		return nil, fmt.Errorf("consume %s: %w", consumerName, err)
	}

	slog.Info("consuming started", "consumer", consumerName)
	return cc, nil
}

// RecreateConsumer deletes the named consumer if it exists, then creates it fresh.
// Useful to enforce DeliverNewPolicy across service restarts.
func (c *Client) RecreateConsumer(ctx context.Context, streamName string, cfg jetstream.ConsumerConfig) error {
	c.mu.RLock()
	stream, ok := c.streams[streamName]
	c.mu.RUnlock()

	if !ok {
		var err error
		stream, err = c.js.Stream(ctx, streamName)
		if err != nil {
			return fmt.Errorf("get stream %s: %w", streamName, err)
		}
	}

	// Delete the consumer if it already exists; ignore not-found.
	if err := stream.DeleteConsumer(ctx, cfg.Name); err != nil && !errors.Is(err, jetstream.ErrConsumerNotFound) {
		return fmt.Errorf("delete consumer %s on stream %s: %w", cfg.Name, streamName, err)
	}

	// Delegate to CreateConsumer — avoids duplicating stream resolution and caching logic.
	if _, err := c.CreateConsumer(ctx, streamName, cfg); err != nil {
		return err
	}

	slog.Info("consumer recreated", "name", cfg.Name, "stream", streamName)
	return nil
}

// Close flushes and closes the NATS connection.
func (c *Client) Close(ctx context.Context) {
	if c.conn != nil {
		_ = c.conn.FlushWithContext(ctx)
		c.conn.Close()
		slog.Info("nats connection closed")
	}
}
