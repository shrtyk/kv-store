package httphandlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
)

type handlersProvider struct {
	store   store.Store
	tl      tlog.TransactionsLogger
	metrics metrics.Metrics
}

func NewHandlersProvider(store store.Store, tl tlog.TransactionsLogger, m metrics.Metrics) *handlersProvider {
	return &handlersProvider{
		store:   store,
		tl:      tl,
		metrics: m,
	}
}

func (h *handlersProvider) HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

func (h *handlersProvider) PutHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	val, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	h.tl.WritePut(key, string(val))
	err = h.store.Put(key, string(val))

	h.metrics.Put(key, time.Since(start).Seconds())

	if err != nil {
		switch {
		case errors.Is(err, store.ErrKeyTooLarge):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, store.ErrValueTooLarge):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *handlersProvider) GetHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	val, err := h.store.Get(key)

	h.metrics.Get(key, time.Since(start).Seconds())

	if err != nil {
		switch {
		case errors.Is(err, store.ErrNoSuchKey):
			http.NotFound(w, r)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if _, err := w.Write([]byte(val)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handlersProvider) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	h.tl.WriteDelete(key)
	err := h.store.Delete(key)

	h.metrics.Delete(key, time.Since(start).Seconds())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
