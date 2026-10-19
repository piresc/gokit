package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/response"
)

const defaultRevocationTimeout = time.Second

const (
	msgAuthRequired = "Authentication required"
	msgTokenExpired = "Token expired"
)

// RS256Options configures RS256Auth and OptionalRS256Auth.
type RS256Options struct {
	// Revocation checks token IDs (jti) against a revocation list. Nil disables the check.
	Revocation auth.RevocationChecker
	// FailOpen lets requests through when the revocation check errors.
	// When false, such requests receive 503.
	FailOpen bool
	// RevocationTimeout bounds each revocation lookup. Zero means 1s.
	RevocationTimeout time.Duration
}

// RS256Auth validates an RS256 Bearer token and stores the claims, user_id
// and role in the request context.
func (m *Middleware) RS256Auth(v *auth.RSAVerifier, opts RS256Options) func(http.Handler) http.Handler {
	return m.rs256(v, opts, false)
}

// OptionalRS256Auth behaves like RS256Auth but lets requests without an
// Authorization header through unauthenticated. A present but invalid header
// is rejected.
func (m *Middleware) OptionalRS256Auth(v *auth.RSAVerifier, opts RS256Options) func(http.Handler) http.Handler {
	return m.rs256(v, opts, true)
}

func (m *Middleware) rs256(v *auth.RSAVerifier, opts RS256Options, optional bool) func(http.Handler) http.Handler {
	timeout := opts.RevocationTimeout
	if timeout <= 0 {
		timeout = defaultRevocationTimeout
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" && optional {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, msgAuthRequired)
				return
			}

			claims, err := v.Verify(strings.TrimSpace(parts[1]))
			if err != nil {
				if errors.Is(err, auth.ErrTokenExpired) {
					response.Error(w, http.StatusUnauthorized, response.CodeTokenExpired, msgTokenExpired)
					return
				}
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, msgAuthRequired)
				return
			}

			if opts.Revocation != nil && claims.ID != "" {
				ctx, cancel := context.WithTimeout(r.Context(), timeout)
				revoked, err := opts.Revocation.IsRevoked(ctx, claims.ID)
				cancel()
				if err != nil {
					if opts.FailOpen {
						m.logger.Warn("token revocation check failed, failing open", "jti", claims.ID, "error", err)
					} else {
						m.logger.Error("token revocation check failed", "jti", claims.ID, "error", err)
						w.Header().Set("Retry-After", "5")
						response.Error(w, http.StatusServiceUnavailable, response.CodeServiceUnavailable, "Service unavailable")
						return
					}
				} else if revoked {
					response.Error(w, http.StatusUnauthorized, response.CodeTokenRevoked, "Token has been revoked")
					return
				}
			}

			ctx := context.WithValue(r.Context(), KeyClaims, claims)
			ctx = context.WithValue(ctx, KeyUserID, claims.Subject)
			if claims.Role != "" {
				ctx = context.WithValue(ctx, KeyRole, claims.Role)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the RS256 claims stored by RS256Auth.
func GetClaims(r *http.Request) (*auth.RSAClaims, bool) {
	c, ok := r.Context().Value(KeyClaims).(*auth.RSAClaims)
	return c, ok && c != nil
}
