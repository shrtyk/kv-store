package httphandlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/cfg"
	ftr "github.com/shrtyk/kv-store/internal/core/ports/futures"
	"github.com/shrtyk/kv-store/internal/core/ports/metrics"
	"github.com/shrtyk/kv-store/internal/core/ports/store"
	"github.com/shrtyk/kv-store/pkg/logger"
	fsm_v1 "github.com/shrtyk/kv-store/proto/fsm/gen"
	raftapi "github.com/shrtyk/raft-core/api"
	"google.golang.org/protobuf/proto"
)

type handlersProvider struct {
	stCfg               *cfg.StoreCfg
	store               store.Store
	metrics             metrics.Metrics
	raft                raftapi.Raft
	futures             ftr.FuturesStore
	raftPublicHTTPAddrs []string
}

func NewHandlersProvider(
	stCfg *cfg.StoreCfg,
	store store.Store,
	m metrics.Metrics,
	raft raftapi.Raft,
	futures ftr.FuturesStore,
	raftPublicHTTPAddrs []string,
) *handlersProvider {
	return &handlersProvider{
		stCfg:               stCfg,
		store:               store,
		metrics:             m,
		raft:                raft,
		futures:             futures,
		raftPublicHTTPAddrs: raftPublicHTTPAddrs,
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
// @Failure 	 307 {string} string "Node is not a leader"
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

	cmd := &fsm_v1.Command{
		Command: &fsm_v1.Command_Put{
			Put: &fsm_v1.PutCommand{
				Key:   key,
				Value: string(val),
			},
		},
	}
	data, err := proto.Marshal(cmd)
	if err != nil {
		http.Error(w, "failed to marshal command", http.StatusInternalServerError)
		return
	}

	res := h.raft.Submit(data)
	if !res.IsLeader {
		h.redirect(w, r.URL.Path, res.LeaderID)
		return
	}

	promise := h.futures.NewFuture(res.LogIndex)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := promise.Wait(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "request timed out: raft cluster is busy", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.metrics.HttpPut(key, time.Since(start).Seconds())
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
// @Failure 	 307 {string} string "Node is not a leader"
// @Failure      404
// @Failure      500 {string} string "Internal Server Error"
// @Router       /v1/{key} [get]
func (h *handlersProvider) GetHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.FromCtx(r.Context())
	start := time.Now()

	key := chi.URLParam(r, "key")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.raft.ReadOnly(ctx, []byte(key))
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNoSuchKey):
			http.NotFound(w, r)
		case errors.Is(err, context.DeadlineExceeded):
			http.Error(w, "request timed out: raft cluster is busy", http.StatusServiceUnavailable)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if !resp.IsLeader {
		h.redirect(w, r.URL.Path, resp.LeaderId)
		return
	}

	if _, err := w.Write(resp.Data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.metrics.HttpGet(key, time.Since(start).Seconds())
	l.Debug(
		"Get operation successfully completed",
		slog.String("key", key),
		slog.String("value", string(resp.Data)))
}

// DeleteHandler godoc
// @Summary      Deletes a value from the store
// @Description  Deletes a value from the store
// @Tags         store
// @Param        key path string true "key"
// @Success      204
// @Failure 	 307 {string} string "Node is not a leader"
// @Failure      500 {string} string "Internal Server Error"
// @Router       /v1/{key} [delete]
func (h *handlersProvider) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.FromCtx(r.Context())
	start := time.Now()

	key := chi.URLParam(r, "key")

	cmd := &fsm_v1.Command{
		Command: &fsm_v1.Command_Delete{
			Delete: &fsm_v1.DeleteCommand{
				Key: key,
			},
		},
	}
	data, err := proto.Marshal(cmd)
	if err != nil {
		http.Error(w, "failed to marshal command", http.StatusInternalServerError)
		return
	}

	res := h.raft.Submit(data)
	if !res.IsLeader {
		h.redirect(w, r.URL.Path, res.LeaderID)
		return
	}

	promise := h.futures.NewFuture(res.LogIndex)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := promise.Wait(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, "request timed out: raft cluster is busy", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	h.metrics.HttpDelete(key, time.Since(start).Seconds())
	l.Debug("Delete operation successfully completed", slog.String("key", key))
}

func (h *handlersProvider) redirect(w http.ResponseWriter, urlPath string, leaderId int) {
	if leaderId >= 0 && leaderId < len(h.raftPublicHTTPAddrs) {
		leaderAddr := h.raftPublicHTTPAddrs[leaderId]
		redirectURL := fmt.Sprintf("%s%s", leaderAddr, urlPath)
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(w, "no leader available", http.StatusServiceUnavailable)
	}
}
