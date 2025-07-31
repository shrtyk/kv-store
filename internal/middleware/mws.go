package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
)

type Middlewares interface {
	HttpMetrics(http.Handler) http.HandlerFunc
}

type mws struct {
	metrics metrics.Metrics
}

func NewMiddlewares(m metrics.Metrics) *mws {
	return &mws{
		metrics: m,
	}
}

type customResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *customResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *customResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (m *mws) HttpMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		cw := &customResponseWriter{ResponseWriter: w}
		next.ServeHTTP(cw, r)

		m.metrics.HttpRequest(
			cw.statusCode,
			r.Method,
			chi.RouteContext(r.Context()).RoutePattern(),
			time.Since(start).Seconds())
	})
}
