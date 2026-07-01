package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Config holds rate limiter configuration.
type Config struct {
	RequestsPerSecond int
	BurstSize         int
	CleanupInterval   time.Duration
}

// DefaultConfig returns sane defaults: 100 req/s, burst 200, cleanup every 5m.
func DefaultConfig() Config {
	return Config{
		RequestsPerSecond: 100,
		BurstSize:         200,
		CleanupInterval:   5 * time.Minute,
	}
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Store is a thread-safe in-memory per-key token-bucket rate limiter store.
type Store struct {
	mu       sync.Mutex
	limiters map[string]*entry
	cfg      Config
	stopCh   chan struct{}
}

// NewStore creates a Store and starts the background cleanup goroutine when
// cfg.CleanupInterval > 0.
func NewStore(cfg Config) *Store {
	s := &Store{
		limiters: make(map[string]*entry),
		cfg:      cfg,
		stopCh:   make(chan struct{}),
	}
	if cfg.CleanupInterval > 0 {
		go s.cleanup(cfg.CleanupInterval)
	}
	return s
}

// Allow returns true if the key is within the rate limit.
func (s *Store) Allow(key string) bool {
	s.mu.Lock()
	e, exists := s.limiters[key]
	if !exists {
		e = &entry{
			limiter:  rate.NewLimiter(rate.Limit(s.cfg.RequestsPerSecond), s.cfg.BurstSize),
			lastSeen: time.Now(),
		}
		s.limiters[key] = e
	} else {
		e.lastSeen = time.Now()
	}
	s.mu.Unlock()
	return e.limiter.Allow()
}

// Stop halts the background cleanup goroutine.
func (s *Store) Stop() {
	close(s.stopCh)
}

func (s *Store) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, e := range s.limiters {
				if now.Sub(e.lastSeen) > 2*interval {
					delete(s.limiters, key)
				}
			}
			s.mu.Unlock()
		case <-s.stopCh:
			return
		}
	}
}
