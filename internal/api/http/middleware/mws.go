package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shrtyk/kv-store/internal/core/ports/metrics"
	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/tomasen/realip"
)

type mws struct {
	log     *slog.Logger
	metrics metrics.Metrics
}

func NewMiddlewares(l *slog.Logger, m metrics.Metrics) *mws {
	return &mws{
		log:     l,
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

		cw := &customResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(cw, r)

		m.metrics.HttpRequest(
			cw.statusCode,
			r.Method,
			chi.RouteContext(r.Context()).RoutePattern(),
			time.Since(start).Seconds())
	})
}

func (m *mws) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxWithLog := logger.ToCtx(r.Context(), m.log.With(
			slog.String("ip", realip.FromRequest(r)),
			slog.String("user-agent", r.UserAgent()),
			slog.String("request_id", uuid.New().String()),
			slog.String("method", r.Method),
			slog.String("url", r.URL.RequestURI()),
		))

		newReq := r.WithContext(ctxWithLog)
		next.ServeHTTP(w, newReq)
	})
}

func (m *mws) RequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
