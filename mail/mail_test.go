package mail_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/mail"
)

func TestNewSender_ReturnsLogSenderWhenNoSMTP(t *testing.T) {
	s := mail.NewSender(config.MailConfig{})
	assert.NotNil(t, s)
	// LogSender.Send should not error
	err := s.Send(context.Background(), "a@b.com", "hi", "body")
	assert.NoError(t, err)
}

func TestNewSender_ReturnsSMTPSenderWhenConfigured(t *testing.T) {
	s := mail.NewSender(config.MailConfig{
		Provider: "smtp",
		SMTPHost: "localhost",
		SMTPPort: 25,
		From:     "from@example.com",
	})
	_, ok := s.(*mail.SMTPSender)
	assert.True(t, ok)
}

func TestLogSender_Send(t *testing.T) {
	s := &mail.LogSender{}
	err := s.Send(context.Background(), "to@example.com", "subject", "body text")
	assert.NoError(t, err)
}

func TestInMemorySender_CapturesMessages(t *testing.T) {
	s := &mail.InMemorySender{}
	require.NoError(t, s.Send(context.Background(), "a@b.com", "sub1", "body1"))
	require.NoError(t, s.Send(context.Background(), "c@d.com", "sub2", "body2"))

	assert.Len(t, s.Messages, 2)
	assert.Equal(t, "a@b.com", s.Messages[0].To)
	assert.Equal(t, "sub1", s.Messages[0].Subject)
	assert.Equal(t, "body1", s.Messages[0].Body)
	assert.Equal(t, "c@d.com", s.Messages[1].To)
}
