package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/config"
	"github.com/piresc/gokit/redis"
)

func newRevocationStore(t *testing.T) (*auth.RedisRevocationStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client, err := redis.NewClient(config.RedisConfig{
		Host: mr.Host(),
		Port: int(mr.Server().Addr().Port),
	})
	if err != nil {
		t.Fatalf("redis.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return auth.NewRedisRevocationStore(client), mr
}

func TestRevocation_RevokeThenIsRevoked(t *testing.T) {
	store, mr := newRevocationStore(t)
	ctx := context.Background()
	ttl := 10 * time.Minute

	if err := store.Revoke(ctx, "j1", ttl); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	revoked, err := store.IsRevoked(ctx, "j1")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("expected j1 to be revoked")
	}
	if !mr.Exists(auth.RevocationKeyPrefix + "j1") {
		t.Fatal("expected key revoked:j1 to exist")
	}
	if got := mr.TTL("revoked:j1"); got <= 0 || got > ttl {
		t.Fatalf("TTL = %v, want in (0, %v]", got, ttl)
	}
}

func TestRevocation_UnknownJTI(t *testing.T) {
	store, _ := newRevocationStore(t)

	revoked, err := store.IsRevoked(context.Background(), "nope")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if revoked {
		t.Fatal("unknown jti must not be revoked")
	}
}

func TestRevocation_ZeroTTLIsNoop(t *testing.T) {
	store, mr := newRevocationStore(t)
	ctx := context.Background()

	if err := store.Revoke(ctx, "j1", 0); err != nil {
		t.Fatalf("Revoke zero ttl: %v", err)
	}
	if err := store.Revoke(ctx, "j2", -time.Second); err != nil {
		t.Fatalf("Revoke negative ttl: %v", err)
	}
	if err := store.Revoke(ctx, "", time.Minute); err != nil {
		t.Fatalf("Revoke empty jti: %v", err)
	}
	if len(mr.Keys()) != 0 {
		t.Fatalf("expected no keys, got %v", mr.Keys())
	}
}

func TestRevocation_ExpiresAfterTTL(t *testing.T) {
	store, mr := newRevocationStore(t)
	ctx := context.Background()
	ttl := time.Minute

	if err := store.Revoke(ctx, "j1", ttl); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	mr.FastForward(ttl + time.Second)

	revoked, err := store.IsRevoked(ctx, "j1")
	if err != nil {
		t.Fatalf("IsRevoked: %v", err)
	}
	if revoked {
		t.Fatal("expected revocation to expire")
	}
}

func TestRevocation_RedisDown(t *testing.T) {
	store, mr := newRevocationStore(t)
	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	revoked, err := store.IsRevoked(ctx, "j1")
	t.Logf("IsRevoked against down redis took %v", time.Since(start))
	if err == nil {
		t.Fatal("expected error when redis is down")
	}
	if revoked {
		t.Fatal("must not report revoked on error")
	}
}
