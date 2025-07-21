package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

func Serve(addr string, handlers http.Handler) {
	s := http.Server{
		Addr:         addr,
		Handler:      handlers,
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

func NewHandler() *chi.Mux {
	mux := chi.NewMux()

	mux.Group(func(r chi.Router) {
		r.HandleFunc("/", helloHandler)
	})

	return mux
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}
