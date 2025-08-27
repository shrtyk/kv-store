package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/shrtyk/kv-store/api/openapi"
	"github.com/shrtyk/kv-store/internal/api/grpc"
	appHttp "github.com/shrtyk/kv-store/internal/api/http"
	mw "github.com/shrtyk/kv-store/internal/api/http/middleware"
	"github.com/shrtyk/kv-store/pkg/logger"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *application) Serve(ctx context.Context, wg *sync.WaitGroup) {
	httpServ := http.Server{
		Addr:         ":" + app.cfg.HttpCfg.Port,
		Handler:      app.NewRouter(),
		IdleTimeout:  app.cfg.HttpCfg.ServerIdleTimeout,
		WriteTimeout: app.cfg.HttpCfg.ServerWriteTimeout,
		ReadTimeout:  app.cfg.HttpCfg.ServerReadTimeout,
	}
	grpcServ := grpc.NewGRPCServer(
		wg,
		&app.cfg.GRPCCfg,
		&app.cfg.Store,
		app.store,
		app.tl,
		app.metrics,
		app.logger,
	)

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()

		tCtx, tCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer tCancel()

		app.logger.Info("got a signal to stop work. executing graceful shutdown")

		errCh <- grpcServ.Shutdown(tCtx)
		errCh <- httpServ.Shutdown(tCtx)
		close(errCh)
	}()

	app.tl.Start(ctx, wg, app.store)
	app.store.StartMapRebuilder(ctx, wg)

	app.logger.Info("grpc listening", slog.String("port", app.cfg.GRPCCfg.Port))
	grpcServ.MustStart()
	app.logger.Info("http listening", slog.String("port", app.cfg.HttpCfg.Port))
	if err := httpServ.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server failed to start: %v", err)
	}

	if err := <-errCh; err != nil {
		app.logger.Error("failed server shutdown", logger.ErrorAttr(err))
		return
	}

	wg.Wait()
	app.logger.Info("application stopped")
}

type HandlersProvider interface {
	Healthz(w http.ResponseWriter, r *http.Request)
	PutHandler(w http.ResponseWriter, r *http.Request)
	GetHandler(w http.ResponseWriter, r *http.Request)
	DeleteHandler(w http.ResponseWriter, r *http.Request)
}

type Middlewares interface {
	HttpMetrics(http.Handler) http.Handler
	Logging(next http.Handler) http.Handler
}

func (app *application) NewRouter() *chi.Mux {
	var handlers HandlersProvider = appHttp.NewHandlersProvider(&app.cfg.Store, app.store, app.tl, app.metrics)
	var mws Middlewares = mw.NewMiddlewares(app.logger, app.metrics)

	mux := chi.NewMux()

	mux.Mount("/debug", chimw.Profiler())

	mux.Handle("/metrics", promhttp.Handler())
	mux.Get("/swagger/*", httpSwagger.WrapHandler)
	mux.Get("/healthz", handlers.Healthz)
	mux.Route("/v1", func(r chi.Router) {
		r.Use(chimw.Recoverer, mws.Logging, mws.HttpMetrics)

		r.Put("/{key}", handlers.PutHandler)
		r.Get("/{key}", handlers.GetHandler)
		r.Delete("/{key}", handlers.DeleteHandler)
	})

	return mux
}
