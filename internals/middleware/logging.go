package middle

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func Logger(log *zerolog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			reqID := middleware.GetReqID(r.Context())

			next.ServeHTTP(w, r)

			log.Info().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Dur("duration", time.Since(start)).
				Msg("request completed")
			
		})
	}
}
