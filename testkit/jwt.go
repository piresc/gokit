package testkit

import (
	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/config"
)

// GenerateToken creates a valid JWT token for test authentication.
// Uses auth.GenerateToken with sensible defaults: 3600-sec expiration, "test" issuer.
// Panics on error since test helpers should not silently fail.
func GenerateToken(userID, role, secret string) string {
	cfg := config.JWTConfig{
		Secret:     secret,
		Expiration: 3600,
		Issuer:     "test",
	}

	token, _, err := auth.GenerateToken(userID, role, cfg)
	if err != nil {
		panic("testkit: GenerateToken failed: " + err.Error())
	}

	return token
}
