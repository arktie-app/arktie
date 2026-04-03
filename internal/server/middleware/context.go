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
