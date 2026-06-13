// Package platform holds cross-cutting infrastructure: logging and error helpers.
package platform

import (
	"context"
	"log/slog"
	"os"
)

// ctxKey is a private context-key type to avoid collisions.
type ctxKey int

const requestIDKey ctxKey = iota

// NewLogger builds a slog.Logger. JSON handler in production, text locally.
// level is one of: debug, info, warn, error (default info).
func NewLogger(level string, prod bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	var h slog.Handler
	if prod {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID stores a request id in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request id, or "" if none.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerFrom returns a request-scoped logger carrying the request id if present.
// Falls back to the given base logger.
func LoggerFrom(ctx context.Context, base *slog.Logger) *slog.Logger {
	if id := RequestIDFromContext(ctx); id != "" {
		return base.With("request_id", id)
	}
	return base
}
