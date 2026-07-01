package nats

import (
	"context"
	"errors"
	"fmt"
)

// ErrNilClient is returned when the publisher has no client.
var ErrNilClient = errors.New("publisher client is nil")

// Publisher wraps Client with a simple Publish method.
type Publisher struct {
	client *Client
}

// NewPublisher creates a Publisher backed by client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish sends data to subject, optionally deduplicating by msgID.
func (p *Publisher) Publish(ctx context.Context, subject string, data []byte, msgID string) error {
	if p.client == nil {
		return ErrNilClient
	}
	_, err := p.client.Publish(ctx, subject, data, msgID)
	if err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	return nil
}
