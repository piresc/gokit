package auth_test

import (
	"testing"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/config"
)

var testCfg = config.JWTConfig{
	Secret:     "test-secret-that-is-at-least-32-chars-long",
	Expiration: 3600,
	Issuer:     "gokit-test",
}

func TestGenerateAndValidate(t *testing.T) {
	token, expiresAt, err := auth.GenerateToken("user-123", "admin", testCfg)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if expiresAt.IsZero() {
		t.Fatal("zero expiry")
	}

	claims, err := auth.ValidateToken(token, testCfg.Secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("want user-123, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("want admin, got %s", claims.Role)
	}
}

func TestValidateToken_InvalidSecret(t *testing.T) {
	token, _, _ := auth.GenerateToken("u1", "user", testCfg)
	_, err := auth.ValidateToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	_, err := auth.ValidateToken("not.a.token", testCfg.Secret)
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}
