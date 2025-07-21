package httphandlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
)

type handlersProvider struct {
	store store.Store
	tl    tlog.TransactionsLogger
}

func NewHandlersProvider(store store.Store, tl tlog.TransactionsLogger) *handlersProvider {
	return &handlersProvider{
		store: store,
		tl:    tl,
	}
}

func (h *handlersProvider) HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}

func (h *handlersProvider) PutHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	val, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	h.tl.WritePut(key, string(val))
	err = h.store.Put(key, string(val))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *handlersProvider) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	val, err := h.store.Get(key)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrorNoSuchKey):
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
	key := chi.URLParam(r, "key")

	h.tl.WriteDelete(key)
	err := h.store.Delete(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
