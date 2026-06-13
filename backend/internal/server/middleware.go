package server

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// requestLogger is a structured slog request logger. It records method, path,
// status, byte count and latency, tagged with the chi request id. No PII (DSGVO §11).
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := chimw.GetReqID(r.Context())

		// Propagate the request id into our platform context for downstream loggers.
		ctx := platform.WithRequestID(r.Context(), reqID)
		r = r.WithContext(ctx)

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.log.Info("http_request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
