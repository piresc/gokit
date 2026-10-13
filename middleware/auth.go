package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/response"
)

const (
	// KeyUserID is the context key for the authenticated user ID.
	KeyUserID contextKey = "user_id"
	// KeyRole is the context key for the authenticated user role.
	KeyRole contextKey = "role"
)

// JWTAuth validates a Bearer token from the Authorization header and stores
// user_id and role in the request context.
func (m *Middleware) JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			claims, err := auth.ValidateToken(parts[1], secret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			if claims.UserID == "" {
				response.Error(w, http.StatusUnauthorized, "token missing user identity")
				return
			}

			ctx := context.WithValue(r.Context(), KeyUserID, claims.UserID)
			if claims.Role != "" {
				ctx = context.WithValue(ctx, KeyRole, claims.Role)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns 403 if the authenticated role is not in the allowed list.
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleStr, _ := r.Context().Value(KeyRole).(string)
			if !allowed[roleStr] {
				response.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID retrieves the authenticated user ID stored by JWTAuth.
func GetUserID(r *http.Request) string {
	v, _ := r.Context().Value(KeyUserID).(string)
	return v
}

// GetRole retrieves the authenticated user role stored by JWTAuth.
func GetRole(r *http.Request) string {
	v, _ := r.Context().Value(KeyRole).(string)
	return v
}
