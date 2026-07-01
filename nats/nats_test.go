package nats_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gonatspkg "github.com/piresc/gokit/nats"
)

func TestNewClient_NoServer(t *testing.T) {
	_, err := gonatspkg.NewClient("nats://127.0.0.1:14222")
	assert.Error(t, err, "expected error when no NATS server is running")
}

func TestNewPublisher_NilClient(t *testing.T) {
	p := gonatspkg.NewPublisher(nil)
	require.NotNil(t, p)
	err := p.Publish(context.Background(), "test.subject", []byte("data"), "")
	assert.ErrorIs(t, err, gonatspkg.ErrNilClient)
}

func TestStreamConfig_ToJetStreamConfig(t *testing.T) {
	sc := gonatspkg.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.*"},
	}
	js := sc.ToJetStreamConfig()
	assert.Equal(t, "ORDERS", js.Name)
	assert.Equal(t, []string{"orders.*"}, js.Subjects)
}

func TestConsumerConfig_ToJetStreamConfig(t *testing.T) {
	cc := gonatspkg.ConsumerConfig{
		Name:          "orders-consumer",
		FilterSubject: "orders.created",
	}
	js := cc.ToJetStreamConfig()
	assert.Equal(t, "orders-consumer", js.Name)
	assert.Equal(t, "orders-consumer", js.Durable)
	assert.Equal(t, "orders.created", js.FilterSubject)
}
