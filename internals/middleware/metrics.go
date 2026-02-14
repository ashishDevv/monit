package middle

import (
	"net/http"
	"time"
)

// This middlware will be used in future to collect metrics

type MetricsRecorder interface {
	Observe(method, path string, duration time.Duration)
}

func Metrics(recorder MetricsRecorder) Middleware {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			recorder.Observe(
				r.Method,
				r.URL.Path,
				time.Since(start),
			)
		}
		return http.HandlerFunc(fn)
	}
}
