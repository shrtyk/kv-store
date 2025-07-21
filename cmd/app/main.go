package main

import (
	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/store"
)

func main() {
	ap := app.NewApp()
	ap.Init(
		app.WithStore(store.NewStore()),
	)

	ap.Serve(":16700")
}
