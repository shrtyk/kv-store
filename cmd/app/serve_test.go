package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	"github.com/shrtyk/kv-store/internal/core/store"
	rmocks "github.com/shrtyk/kv-store/internal/core/raft/mocks"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	l, _ := tu.NewMockLogger()

	st := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)
	m := pmts.NewMockMetrics()
	stubRaft := rmocks.NewStubRaft(st, true, 0)
	mockFutures := futuresmocks.NewMockFuturesStore(t)

	app := NewApp()
	app.Init(
		WithCfg(&cfg.AppConfig{}),
		WithLogger(l),
		WithMetrics(m),
		WithStore(st),
		WithRaft(stubRaft),
		WithFutures(mockFutures),
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
