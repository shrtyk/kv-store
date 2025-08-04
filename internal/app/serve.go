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
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/shrtyk/kv-store/api/http"
	mw "github.com/shrtyk/kv-store/internal/middleware"
	transport "github.com/shrtyk/kv-store/internal/transport/http"
	"github.com/shrtyk/kv-store/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) Serve(addr string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	s := http.Server{
		Addr:         addr,
		Handler:      app.NewRouter(),
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
	PutHandler(w http.ResponseWriter, r *http.Request)
	GetHandler(w http.ResponseWriter, r *http.Request)
	DeleteHandler(w http.ResponseWriter, r *http.Request)
}

type Middlewares interface {
	HttpMetrics(http.Handler) http.Handler
	Logging(next http.Handler) http.Handler
}

func (app *application) NewRouter() *chi.Mux {
	var handlers HandlersProvider = transport.NewHandlersProvider(&app.cfg.Store, app.store, app.tl, app.metrics)
	var mws Middlewares = mw.NewMiddlewares(app.logger, app.metrics)

	mux := chi.NewMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.Get("/swagger/*", httpSwagger.WrapHandler)
	mux.Route("/v1", func(r chi.Router) {
		r.Use(chimw.Recoverer, mws.Logging, mws.HttpMetrics)

		r.Put("/{key}", handlers.PutHandler)
		r.Get("/{key}", handlers.GetHandler)
		r.Delete("/{key}", handlers.DeleteHandler)
	})

	return mux
}
