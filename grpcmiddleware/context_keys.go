package grpcmiddleware

import "context"

// contextKey is an unexported type for context value keys to avoid collisions.
type contextKey string

const (
	CtxKeyUserID contextKey = "user_id"
	CtxKeyRole   contextKey = "role"
)

// UserIDFromContext extracts the user ID stored by AuthUnaryInterceptor.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyUserID).(string)
	return v
}

// RoleFromContext extracts the role stored by AuthUnaryInterceptor.
func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CtxKeyRole).(string)
	return v
}

// WithUserID returns a child context with the given user ID.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyUserID, id)
}

// WithRole returns a child context with the given role.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, CtxKeyRole, role)
}
