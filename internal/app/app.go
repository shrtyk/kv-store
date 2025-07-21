package app

import "github.com/shrtyk/kv-store/internal/store"

type application struct {
	store store.Store
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

func WithStore(store store.Store) opt {
	return func(app *application) {
		app.store = store
	}
}
