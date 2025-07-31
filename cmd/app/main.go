package main

import (
	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
)

func main() {
	cfg := cfg.ReadConfig()
	logger := logger.NewLogger(cfg.Env)

	tl := tlog.MustCreateNewFileTransLog(&cfg.TransLogger, logger)
	defer tl.Close()

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
