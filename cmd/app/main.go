package main

import (
	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
)

func main() {
	ap := app.NewApp()
	tl := tlog.MustCreateNewFileTransLog("tlog")
	st := store.NewStore(tl)
	ap.Init(
		app.WithStore(st),
		app.WithTransactionalLogger(tl),
	)

	ap.Serve(":16700")
}
