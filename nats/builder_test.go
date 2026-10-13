package nats_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	gonatspkg "github.com/piresc/gokit/nats"
)

func TestStreamConfigBuilder_Defaults(t *testing.T) {
	cfg := gonatspkg.NewStreamConfigBuilder("MY_STREAM").Build()
	assert.Equal(t, "MY_STREAM", cfg.Name)
	assert.Equal(t, jetstream.LimitsPolicy, cfg.Retention)
	assert.Equal(t, jetstream.FileStorage, cfg.Storage)
	assert.Equal(t, 1, cfg.Replicas)
	assert.Equal(t, 24*time.Hour, cfg.MaxAge)
}

func TestStreamConfigBuilder_WithSubjects(t *testing.T) {
	cfg := gonatspkg.NewStreamConfigBuilder("S").
		WithSubjects("a.b", "c.d").
		Build()
	assert.Equal(t, []string{"a.b", "c.d"}, cfg.Subjects)
}

func TestStreamConfigBuilder_WithRetention(t *testing.T) {
	cfg := gonatspkg.NewStreamConfigBuilder("S").
		WithRetention(jetstream.InterestPolicy).
		Build()
	assert.Equal(t, jetstream.InterestPolicy, cfg.Retention)
}

func TestStreamConfigBuilder_WithStorage(t *testing.T) {
	cfg := gonatspkg.NewStreamConfigBuilder("S").
		WithStorage(jetstream.MemoryStorage).
		Build()
	assert.Equal(t, jetstream.MemoryStorage, cfg.Storage)
}

func TestStreamConfigBuilder_WithMaxAge(t *testing.T) {
	cfg := gonatspkg.NewStreamConfigBuilder("S").
		WithMaxAge(2 * time.Hour).
		Build()
	assert.Equal(t, 2*time.Hour, cfg.MaxAge)
}

func TestConsumerConfigBuilder_Defaults(t *testing.T) {
	cfg := gonatspkg.NewConsumerConfigBuilder("MY_STREAM", "my_consumer").Build()
	assert.Equal(t, "my_consumer", cfg.Name)
	assert.Equal(t, jetstream.DeliverNewPolicy, cfg.DeliverPolicy)
	assert.Equal(t, jetstream.AckExplicitPolicy, cfg.AckPolicy)
	assert.Equal(t, 30*time.Second, cfg.AckWait)
	assert.Equal(t, 3, cfg.MaxDeliver)
}

func TestConsumerConfigBuilder_WithSubject(t *testing.T) {
	cfg := gonatspkg.NewConsumerConfigBuilder("S", "C").
		WithSubject("a.b").
		Build()
	assert.Equal(t, "a.b", cfg.FilterSubject)
}

func TestConsumerConfigBuilder_Chaining(t *testing.T) {
	cfg := gonatspkg.NewConsumerConfigBuilder("S", "C").
		WithDeliverPolicy(jetstream.DeliverAllPolicy).
		WithAckPolicy(jetstream.AckNonePolicy).
		WithMaxDeliver(5).
		WithAckWait(60 * time.Second).
		Build()
	assert.Equal(t, jetstream.DeliverAllPolicy, cfg.DeliverPolicy)
	assert.Equal(t, jetstream.AckNonePolicy, cfg.AckPolicy)
	assert.Equal(t, 5, cfg.MaxDeliver)
	assert.Equal(t, 60*time.Second, cfg.AckWait)
}
