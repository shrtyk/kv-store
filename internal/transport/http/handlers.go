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
	"github.com/shrtyk/kv-store/pkg/cfg"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
)

type handlersProvider struct {
	stCfg   *cfg.StoreCfg
	store   store.Store
	tl      tlog.TransactionsLogger
	metrics metrics.Metrics
}

func NewHandlersProvider(
	stCfg *cfg.StoreCfg,
	store store.Store,
	tl tlog.TransactionsLogger,
	m metrics.Metrics,
) *handlersProvider {
	return &handlersProvider{
		stCfg:   stCfg,
		store:   store,
		tl:      tl,
		metrics: m,
	}
}

func (h *handlersProvider) HelloHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprintln(w, "Hello!"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handlersProvider) PutHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	val, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()

	if len(key) > h.stCfg.MaxKeySize {
		http.Error(w, store.ErrKeyTooLarge.Error(), http.StatusBadRequest)
		return
	}
	if len(val) > h.stCfg.MaxValSize {
		http.Error(w, store.ErrValueTooLarge.Error(), http.StatusBadRequest)
		return
	}

	h.tl.WritePut(key, string(val))
	err = h.store.Put(key, string(val))

	if err != nil {
		switch {
		// First two checks are pretty redundant and left just in case something went terribly wrong
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

	h.metrics.Put(key, time.Since(start).Seconds())
}

func (h *handlersProvider) GetHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	val, err := h.store.Get(key)

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

	h.metrics.Get(key, time.Since(start).Seconds())
}

func (h *handlersProvider) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	key := chi.URLParam(r, "key")
	h.tl.WriteDelete(key)
	err := h.store.Delete(key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.metrics.Delete(key, time.Since(start).Seconds())
}
