package httphandlers

import (
	"fmt"
	"net/http"

	"github.com/shrtyk/kv-store/internal/store"
)

type handlersProvider struct {
	store store.Store
}

func NewHandlersProvider(store store.Store) *handlersProvider {
	return &handlersProvider{
		store: store,
	}
}

func (h handlersProvider) HelloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello!")
}
