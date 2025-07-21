package app

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	httphandlers "github.com/shrtyk/kv-store/internal/transport/http"
)

func (app *application) Serve(addr string) {
	s := http.Server{
		Addr:         addr,
		Handler:      NewRouter(app.store),
		IdleTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errc := make(chan error, 1)
	go func() {
		<-ctx.Done()

		tCtx, tCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer tCancel()

		errc <- s.Shutdown(tCtx)
		close(errc)
	}()

	log.Printf("Listening '%s'\n", addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed && err != nil {
		log.Printf("Error during server start: %v", err)
		return
	}

	if err := <-errc; err != nil {
		log.Printf("Error during server shutdown: %v", err)
		return
	}
}

type HandlersProvider interface {
	HelloHandler(w http.ResponseWriter, r *http.Request)
}

func NewRouter(store store.Store) *chi.Mux {
	handlers := httphandlers.NewHandlersProvider(store)

	mux := chi.NewMux()
	mux.Group(func(r chi.Router) {
		r.HandleFunc("/", handlers.HelloHandler)
	})

	return mux
}
