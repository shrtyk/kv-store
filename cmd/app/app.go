package main

import (
	"log/slog"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/ports/metrics"
		"github.com/shrtyk/kv-store/internal/core/ports/store"
	"github.com/shrtyk/kv-store/internal/core/ports/tlog"
)

type application struct {
	cfg     *cfg.AppConfig
	store   store.Store
	tl      tlog.TransactionsLogger
	logger  *slog.Logger
	metrics metrics.Metrics
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

func WithTransactionalLogger(tl tlog.TransactionsLogger) opt {
	return func(app *application) {
		app.tl = tl
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
