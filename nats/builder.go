package nats

import (
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// StreamConfigBuilder builds a jetstream.StreamConfig with sensible defaults.
type StreamConfigBuilder struct {
	cfg jetstream.StreamConfig
}

// NewStreamConfigBuilder returns a builder initialised with defaults:
// Retention=LimitsPolicy, Storage=FileStorage, Replicas=1, MaxAge=24h.
func NewStreamConfigBuilder(name string) *StreamConfigBuilder {
	return &StreamConfigBuilder{
		cfg: jetstream.StreamConfig{
			Name:      name,
			Retention: jetstream.LimitsPolicy,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
			MaxAge:    24 * time.Hour,
		},
	}
}

// WithSubjects sets the subjects the stream listens on.
func (b *StreamConfigBuilder) WithSubjects(subjects ...string) *StreamConfigBuilder {
	b.cfg.Subjects = subjects
	return b
}

// WithRetention overrides the retention policy.
func (b *StreamConfigBuilder) WithRetention(r jetstream.RetentionPolicy) *StreamConfigBuilder {
	b.cfg.Retention = r
	return b
}

// WithStorage overrides the storage type.
func (b *StreamConfigBuilder) WithStorage(s jetstream.StorageType) *StreamConfigBuilder {
	b.cfg.Storage = s
	return b
}

// WithMaxAge overrides how long messages are kept.
func (b *StreamConfigBuilder) WithMaxAge(d time.Duration) *StreamConfigBuilder {
	b.cfg.MaxAge = d
	return b
}

// Build returns the assembled jetstream.StreamConfig.
func (b *StreamConfigBuilder) Build() jetstream.StreamConfig {
	return b.cfg
}

// ConsumerConfigBuilder builds a jetstream.ConsumerConfig with sensible defaults.
type ConsumerConfigBuilder struct {
	cfg jetstream.ConsumerConfig
}

// NewConsumerConfigBuilder returns a builder for consumerName on streamName with defaults:
// DeliverPolicy=DeliverNewPolicy, AckPolicy=AckExplicitPolicy, AckWait=30s, MaxDeliver=3.
// streamName is accepted for symmetry with the NATS API but is not stored in the config.
func NewConsumerConfigBuilder(_, consumerName string) *ConsumerConfigBuilder {
	return &ConsumerConfigBuilder{
		cfg: jetstream.ConsumerConfig{
			Name:          consumerName,
			DeliverPolicy: jetstream.DeliverNewPolicy,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       30 * time.Second,
			MaxDeliver:    3,
		},
	}
}

// WithSubject sets the filter subject for the consumer.
func (b *ConsumerConfigBuilder) WithSubject(subject string) *ConsumerConfigBuilder {
	b.cfg.FilterSubject = subject
	return b
}

// WithDeliverPolicy overrides the deliver policy.
func (b *ConsumerConfigBuilder) WithDeliverPolicy(p jetstream.DeliverPolicy) *ConsumerConfigBuilder {
	b.cfg.DeliverPolicy = p
	return b
}

// WithAckPolicy overrides the acknowledgment policy.
func (b *ConsumerConfigBuilder) WithAckPolicy(p jetstream.AckPolicy) *ConsumerConfigBuilder {
	b.cfg.AckPolicy = p
	return b
}

// WithAckWait overrides how long the server waits for an ack before redelivering.
func (b *ConsumerConfigBuilder) WithAckWait(d time.Duration) *ConsumerConfigBuilder {
	b.cfg.AckWait = d
	return b
}

// WithMaxDeliver overrides the maximum number of delivery attempts.
func (b *ConsumerConfigBuilder) WithMaxDeliver(n int) *ConsumerConfigBuilder {
	b.cfg.MaxDeliver = n
	return b
}

// Build returns the assembled jetstream.ConsumerConfig.
func (b *ConsumerConfigBuilder) Build() jetstream.ConsumerConfig {
	return b.cfg
}
