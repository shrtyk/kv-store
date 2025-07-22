package main

import (
	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
)

func main() {
	cfg := cfg.ReadConfig()
	logger := logger.NewLogger(cfg.Env)

	tl := tlog.MustCreateNewFileTransLog("transaction.log")
	defer tl.Close()

	st := store.NewStore()

	ap := app.NewApp()
	ap.Init(
		app.WithStore(st),
		app.WithTransactionalLogger(tl),
		app.WithLogger(logger),
	)

	ap.Serve(":16700")

}
