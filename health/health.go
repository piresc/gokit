package health

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	KeyStatus       = "status"
)

// Checker is implemented by any dependency that can report its health.
type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

// Service holds a collection of named health checkers.
type Service struct {
	checkers map[string]Checker
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewService constructs a Service with no checkers registered.
func NewService(logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		checkers: make(map[string]Checker),
		logger:   logger,
	}
}

// AddChecker registers a checker under the given name.
func (s *Service) AddChecker(name string, checker Checker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkers[name] = checker
}

// CheckAll runs all registered checkers concurrently (each with a 5s timeout)
// and returns an aggregated result map.
func (s *Service) CheckAll(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{})
	dependencies := make(map[string]interface{})
	allHealthy := true

	for name, checker := range s.checkers {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("health check cancelled: %w", err)
		}
		depResult := s.runChecker(ctx, name, checker)
		dependencies[name] = depResult
		if depResult[KeyStatus] != StatusHealthy {
			allHealthy = false
		}
	}

	if allHealthy {
		result[KeyStatus] = StatusHealthy
	} else {
		result[KeyStatus] = StatusUnhealthy
	}
	result["dependencies"] = dependencies

	return result, nil
}

func (s *Service) runChecker(ctx context.Context, name string, checker Checker) map[string]interface{} {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := checker.Check(checkCtx)
	duration := time.Since(start)

	depResult := map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
	}

	if err != nil {
		depResult[KeyStatus] = StatusUnhealthy
		depResult["error"] = err.Error()
		s.logger.Error("health check failed",
			slog.String("checker", name),
			slog.String("error", err.Error()),
			slog.Int64("duration_ms", duration.Milliseconds()))
	} else {
		depResult[KeyStatus] = StatusHealthy
	}

	return depResult
}
