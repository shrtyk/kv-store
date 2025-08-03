package main

import (
	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	log "github.com/shrtyk/kv-store/pkg/logger"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
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

	m := metrics.NewPrometheusMetrics()
	st := store.NewStore(&cfg.Store, logger)

	ap := app.NewApp()
	ap.Init(
		app.WithCfg(cfg),
		app.WithStore(st),
		app.WithTransactionalLogger(tl),
		app.WithLogger(logger),
		app.WithMetrics(m),
	)

	ap.Serve(":16700")
}
