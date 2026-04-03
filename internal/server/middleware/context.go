package middleware

import (
	"context"

	"arktie.org/internal/lib/libjwt"
)

// ContextWithUser returns a context with the given claim attached.
// This is intended for use in tests that need to simulate an authenticated user.
func ContextWithUser(ctx context.Context, claim *libjwt.Claim) context.Context {
	return context.WithValue(ctx, ctxKey{}, claim)
}

// UserFromContext returns the JWT claim stored in the context by the User
// middleware. Returns nil if no valid token was present.
func UserFromContext(ctx context.Context) *libjwt.Claim {
	claim, _ := ctx.Value(ctxKey{}).(*libjwt.Claim)
	return claim
}
