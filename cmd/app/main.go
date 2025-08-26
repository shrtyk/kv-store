package main

import (
	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/snapshot"
	"github.com/shrtyk/kv-store/internal/core/store"
	"github.com/shrtyk/kv-store/internal/core/tlog"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	log "github.com/shrtyk/kv-store/pkg/logger"
)

// @title           KV-Store API
// @version         1.0
// @description     A simple key-value store.
func main() {
	cfg := cfg.ReadConfig()
	logger := log.NewLogger(cfg.Env)

	snapshotter := snapshot.NewFileSnapshotter(&cfg.Snapshots, logger)

	tl := tlog.MustCreateNewFileTransLog(&cfg.Wal, logger, snapshotter)
	defer func() {
		if err := tl.Close(); err != nil {
			logger.Error("failed to close transaction logger", log.ErrorAttr(err))
		}
	}()

	m := pmts.NewPrometheusMetrics()
	st := store.NewStore(&cfg.Store, logger)

	ap := NewApp()
	ap.Init(
		WithCfg(cfg),
		WithStore(st),
		WithTransactionalLogger(tl),
		WithLogger(logger),
		WithMetrics(m),
	)

	ap.Serve()
}
