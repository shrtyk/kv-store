package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/store"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	l, _ := tu.NewMockLogger()

	store := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)
	m := pmts.NewMockMetrics()

	app := NewApp()
	app.Init(
		WithCfg(&cfg.AppConfig{}),
		WithLogger(l),
		WithMetrics(m),
		WithStore(store),
	)

	router := app.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	b, err := io.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, "404 page not found\n", string(b))
}
