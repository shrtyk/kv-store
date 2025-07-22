package app

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	httphandlers "github.com/shrtyk/kv-store/internal/transport/http"
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

		log.Println("Got a signal to stop work. Executing graceful shutdown...")
		errc <- s.Shutdown(tCtx)
		close(errc)
	}()

	app.tl.Start(ctx, &wg, app.store)
	log.Printf("Listening '%s'\n", addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed && err != nil {
		log.Printf("Error during server start: %v", err)
		return
	}

	if err := <-errc; err != nil {
		log.Printf("Error during server shutdown: %v", err)
		return
	}
	wg.Wait()
	log.Println("Server stoped")
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
