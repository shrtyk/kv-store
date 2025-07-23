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
	httphandlers "github.com/shrtyk/kv-store/internal/transport/http"
	"github.com/shrtyk/kv-store/pkg/logger"
)

func (app *application) Serve(addr string) {
	s := http.Server{
		Addr:         addr,
		Handler:      NewRouter(app.store, app.tl),
		IdleTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	var wg sync.WaitGroup
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errc := make(chan error, 1)
	go func() {
		<-ctx.Done()

		tCtx, tCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer tCancel()

		app.logger.Info("got a signal to stop work. executing graceful shutdown")
		errc <- s.Shutdown(tCtx)
		close(errc)
	}()

	app.tl.Start(ctx, &wg, app.store)
	app.store.StartMapRebuilder(ctx, &wg)

	app.logger.Info("listening", slog.String("addr", addr))
	if err := s.ListenAndServe(); err != http.ErrServerClosed && err != nil {
		log.Printf("Error during server start: %v", err)
		return
	}

	if err := <-errc; err != nil {
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

func NewRouter(store store.Store, tl tlog.TransactionsLogger) *chi.Mux {
	var handlers HandlersProvider = httphandlers.NewHandlersProvider(store, tl)

	mux := chi.NewMux()
	mux.Route("/v1", func(r chi.Router) {
		r.Get("/", handlers.HelloHandler)

		r.Put("/{key}", handlers.PutHandler)
		r.Get("/{key}", handlers.GetHandler)
		r.Delete("/{key}", handlers.DeleteHandler)
	})

	return mux
}
