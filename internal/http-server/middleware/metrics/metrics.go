package metrics

import (
	"net/http"
	"strconv"
	"time"
	"url-shortener/internal/lib/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			metrics.HTTPRequestsInFlight.Inc()
			defer metrics.HTTPRequestsInFlight.Dec()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				duration := time.Since(start).Seconds()

				routePattern := chi.RouteContext(r.Context()).RoutePattern()
				if routePattern == "" {
					routePattern = "unknown"
				}

				statusCode := strconv.Itoa(ww.Status())

				metrics.HTTPRequestsTotal.WithLabelValues(
					r.Method,
					routePattern,
					statusCode,
				).Inc()

				metrics.HTTPRequestDuration.WithLabelValues(
					r.Method,
					routePattern,
				).Observe(duration)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
