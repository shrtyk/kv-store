package app

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	transport "github.com/shrtyk/kv-store/internal/transport/http"
	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/shrtyk/kv-store/pkg/otel"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (app *application) Serve(addr string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	promHandler, shutdown, err := otel.OTelMetrics(ctx, otel.MustCreatePrometheusExporter())
	if err != nil {
		app.logger.Error("failed to setup OTel SDK", logger.ErrorAttr(err))
		return
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			app.logger.Error("failed to shutdown OTel SDK", logger.ErrorAttr(err))
		}
	}()

	s := http.Server{
		Addr:         addr,
		Handler:      NewRouter(app.store, app.tl, promHandler),
		IdleTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	var wg sync.WaitGroup

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		tCtx, tCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer tCancel()

		app.logger.Info("got a signal to stop work. executing graceful shutdown")
		errCh <- s.Shutdown(tCtx)
		close(errCh)
	}()

	app.tl.Start(ctx, &wg, app.store)
	app.tl.Compact()
	app.store.StartMapRebuilder(ctx, &wg)

	app.logger.Info("listening", slog.String("addr", addr))
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server failed to start: %v", err)
	}

	if err := <-errCh; err != nil {
		app.logger.Error("failed server shutdown", logger.ErrorAttr(err))
		return
	}

	wg.Wait()
	app.logger.Info("server stopped")
}

type HandlersProvider interface {
	HelloHandler(w http.ResponseWriter, r *http.Request)
	PutHandler(w http.ResponseWriter, r *http.Request)
	GetHandler(w http.ResponseWriter, r *http.Request)
	DeleteHandler(w http.ResponseWriter, r *http.Request)
}

func NewRouter(
	store store.Store,
	tl tlog.TransactionsLogger,
	promHandler http.Handler,
) *chi.Mux {
	var handlers HandlersProvider = transport.NewHandlersProvider(store, tl)

	mux := chi.NewMux()
	mux.Use(otelhttp.NewMiddleware("http"))
	mux.Handle("/metrics", promHandler)
	mux.Route("/v1", func(r chi.Router) {
		r.Get("/", handlers.HelloHandler)

		r.Put("/{key}", handlers.PutHandler)
		r.Get("/{key}", handlers.GetHandler)
		r.Delete("/{key}", handlers.DeleteHandler)
	})

	return mux
}
