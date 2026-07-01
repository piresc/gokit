package middleware

import (
	"net/http"
	"strings"

	"github.com/piresc/gokit/auth"
	"github.com/piresc/gokit/response"

	"github.com/gin-gonic/gin"
)

const (
	// KeyUserID is the gin.Context key for the authenticated user ID.
	KeyUserID = "user_id"
	// KeyRole is the gin.Context key for the authenticated user role.
	KeyRole = "role"
)

// JWTAuth validates a Bearer token from the Authorization header and stores
// user_id and role in the Gin context.
func (m *Middleware) JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if claims.UserID == "" {
			c.Abort()
			response.Error(c, http.StatusUnauthorized, "token missing user identity")
			return
		}

		c.Set(KeyUserID, claims.UserID)
		if claims.Role != "" {
			c.Set(KeyRole, claims.Role)
		}

		c.Next()
	}
}

// RequireRole aborts with 403 if the authenticated role is not in the allowed list.
func (m *Middleware) RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(KeyRole)
		roleStr, _ := role.(string)
		if !allowed[roleStr] {
			c.Abort()
			response.Error(c, http.StatusForbidden, "insufficient permissions")
			return
		}
		c.Next()
	}
}
