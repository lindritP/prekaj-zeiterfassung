package server

import (
	"context"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/auth"
)

type ctxKey int

const identityKey ctxKey = iota

// withIdentity stores the authenticated principal in the context.
func withIdentity(ctx context.Context, id auth.Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// identityFrom returns the principal placed by requireAuth, if present.
func identityFrom(ctx context.Context) (auth.Identity, bool) {
	id, ok := ctx.Value(identityKey).(auth.Identity)
	return id, ok
}
