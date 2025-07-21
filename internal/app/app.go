package app

import (
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
)

type application struct {
	store store.Store
	tl    tlog.TransactionsLogger
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

func WithTransactionalLogger(tl tlog.TransactionsLogger) opt {
	return func(app *application) {
		app.tl = tl
	}
}
