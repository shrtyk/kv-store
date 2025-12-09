package main

import (
	"log/slog"

	"github.com/shrtyk/kv-store/internal/cfg"
	ftr "github.com/shrtyk/kv-store/internal/core/ports/futures"
	"github.com/shrtyk/kv-store/internal/core/ports/metrics"
	"github.com/shrtyk/kv-store/internal/core/ports/store"
	raftapi "github.com/shrtyk/raft-core/api"
)

type application struct {
	cfg                 *cfg.AppConfig
	store               store.Store
	logger              *slog.Logger
	metrics             metrics.Metrics
	raft                raftapi.Raft
	futures             ftr.FuturesStore
	raftPublicHTTPAddrs []string
}

type opt func(*application)

func NewApp() *application {
	return &application{}
}

func (app *application) Init(opts ...opt) {
	for _, op := range opts {
		op(app)
	}
}

func WithCfg(cfg *cfg.AppConfig) opt {
	return func(app *application) {
		app.cfg = cfg
	}
}

func WithStore(store store.Store) opt {
	return func(app *application) {
		app.store = store
	}
}

func WithLogger(l *slog.Logger) opt {
	return func(app *application) {
		app.logger = l
	}
}

func WithMetrics(m metrics.Metrics) opt {
	return func(app *application) {
		app.metrics = m
	}
}

func WithRaft(r raftapi.Raft) opt {
	return func(app *application) {
		app.raft = r
	}
}

func WithFutures(f ftr.FuturesStore) opt {
	return func(app *application) {
		app.futures = f
	}
}

func WithRaftPublicHTTPAddrs(addrs []string) opt {
	return func(app *application) {
		app.raftPublicHTTPAddrs = addrs
	}
}
