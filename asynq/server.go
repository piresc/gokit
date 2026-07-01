package asynq

import (
	"fmt"

	"github.com/hibiken/asynq"
)

const defaultConcurrency = 10

// Server wraps asynq.Server with a handler mux.
type Server struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

// NewServer creates a Server connected to redisAddr with optional password and concurrency.
func NewServer(redisAddr, redisPassword string, concurrency int) *Server {
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: concurrency,
	})
	return &Server{server: srv, mux: asynq.NewServeMux()}
}

// Register registers a handler for taskType.
func (s *Server) Register(taskType string, handler asynq.Handler) {
	s.mux.Handle(taskType, handler)
}

// Start begins processing tasks.
func (s *Server) Start() error {
	if err := s.server.Start(s.mux); err != nil {
		return fmt.Errorf("start asynq server: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	s.server.Shutdown()
}
