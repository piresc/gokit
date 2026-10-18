package nats_test

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	gonatspkg "github.com/piresc/gokit/nats"
)

type received struct {
	ctx context.Context
	msg jetstream.Msg
}

// setupTracing starts an in-process JetStream server, installs a recording
// tracer provider and W3C propagator, and returns a client with stream T and
// consumer c ready.
func setupTracing(t *testing.T) (*gonatspkg.Client, *tracetest.SpanRecorder) {
	t.Helper()

	opts := natsserver.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()
	srv := natsserver.RunServer(&opts)
	t.Cleanup(srv.Shutdown)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prevTP := otel.GetTracerProvider()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
		_ = tp.Shutdown(context.Background())
	})

	client, err := gonatspkg.NewClient(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(func() { client.Close(context.Background()) })

	ctx := context.Background()
	_, err = client.CreateStream(ctx, jetstream.StreamConfig{Name: "T", Subjects: []string{"t.*"}})
	require.NoError(t, err)
	_, err = client.CreateConsumer(ctx, "T", jetstream.ConsumerConfig{Name: "c", Durable: "c"})
	require.NoError(t, err)

	return client, recorder
}

func consumeOne(t *testing.T, client *gonatspkg.Client) <-chan received {
	t.Helper()
	ch := make(chan received, 1)
	cc, err := client.ConsumeContext("c", func(ctx context.Context, msg jetstream.Msg) {
		ch <- received{ctx: ctx, msg: msg}
		_ = msg.Ack()
	})
	require.NoError(t, err)
	t.Cleanup(cc.Stop)
	return ch
}

func TestPublish_PropagatesTraceToConsumer(t *testing.T) {
	client, recorder := setupTracing(t)
	ch := consumeOne(t, client)

	ctx, root := otel.Tracer("test").Start(context.Background(), "root")
	_, err := client.Publish(ctx, "t.a", []byte("payload"), "")
	require.NoError(t, err)
	root.End()

	var got received
	select {
	case got = <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	wantTrace := root.SpanContext().TraceID()
	assert.Equal(t, wantTrace, trace.SpanContextFromContext(got.ctx).TraceID())
	assert.NotEmpty(t, got.msg.Headers().Get("traceparent"))

	require.Eventually(t, func() bool {
		var pub, proc bool
		for _, s := range recorder.Ended() {
			if s.SpanContext().TraceID() != wantTrace {
				continue
			}
			switch s.Name() {
			case "t.a publish":
				pub = s.SpanKind() == trace.SpanKindProducer
			case "t.a process":
				proc = s.SpanKind() == trace.SpanKindConsumer
			}
		}
		return pub && proc
	}, 2*time.Second, 10*time.Millisecond)
}

func TestPublish_NoParentSpan(t *testing.T) {
	client, _ := setupTracing(t)
	ch := consumeOne(t, client)

	_, err := client.Publish(context.Background(), "t.b", []byte("payload"), "")
	require.NoError(t, err)

	select {
	case got := <-ch:
		assert.Equal(t, "", got.msg.Headers().Get("traceparent"))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestConsumeContext_UnknownConsumer(t *testing.T) {
	client, _ := setupTracing(t)

	_, err := client.ConsumeContext("missing", func(context.Context, jetstream.Msg) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
