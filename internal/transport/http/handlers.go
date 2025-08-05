package httphandlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
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

// Healthz godoc
// @Summary      Healthz
// @Description  Health check
// @Tags         store
// @Produce      text/plain
// @Success      200 {string} string
// @Failure      500 {string} string "Internal Server Error"
// @Router       /healthz [get]
func (h *handlersProvider) Healthz(w http.ResponseWriter, r *http.Request) {
	if _, err := fmt.Fprint(w, "kv-store up and healthy"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// PutHandler godoc
// @Summary      Puts a value into the store
// @Description  Puts a value into the store
// @Tags         store
// @Accept       text/plain
// @Produce      text/plain
// @Param        key path string true "key"
// @Param        value body string true "value"
// @Success      201
// @Failure      400 {string} string "Wrong input data"
// @Failure      500 {string} string "Internal Server Error"
// @Router       /v1/{key} [put]
func (h *handlersProvider) PutHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.FromCtx(r.Context())
	start := time.Now()

	key := chi.URLParam(r, "key")
	val, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			l.Error("failed to close request body", logger.ErrorAttr(err))
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
	l.Debug(
		"Put operation successfully completed",
		slog.String("key", key),
		slog.String("value", string(val)))
}

// GetHandler godoc
// @Summary      Gets a value from the store
// @Description  Gets a value from the store
// @Tags         store
// @Produce      text/plain
// @Param        key path string true "key"
// @Success      200 {string} string "value"
// @Failure      404
// @Failure      500 {string} string "Internal Server Error"
// @Router       /v1/{key} [get]
func (h *handlersProvider) GetHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.FromCtx(r.Context())
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
	l.Debug(
		"Get operation successfully completed",
		slog.String("key", key),
		slog.String("value", val))
}

// DeleteHandler godoc
// @Summary      Deletes a value from the store
// @Description  Deletes a value from the store
// @Tags         store
// @Param        key path string true "key"
// @Success      204
// @Failure      500 {string} string "Internal Server Error"
// @Router       /v1/{key} [delete]
func (h *handlersProvider) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.FromCtx(r.Context())
	start := time.Now()

	key := chi.URLParam(r, "key")
	h.tl.WriteDelete(key)
	err := h.store.Delete(key)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.metrics.Delete(key, time.Since(start).Seconds())
	l.Debug("Delete operation successfully completed", slog.String("key", key))
}
