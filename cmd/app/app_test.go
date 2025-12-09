package main

import (
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	"github.com/shrtyk/kv-store/internal/core/store"
	rmocks "github.com/shrtyk/kv-store/internal/core/raft/mocks"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	appCfg := &cfg.AppConfig{}
	l, _ := tu.NewMockLogger()
	st := store.NewStore(&sync.WaitGroup{}, &cfg.StoreCfg{}, &cfg.ShardsCfg{}, l)
	metrics := pmts.NewMockMetrics()
	stubRaft := rmocks.NewStubRaft(st, true, 0)
	mockFutures := futuresmocks.NewMockFuturesStore(t)

	tapp := NewApp()
	tapp.Init(
		WithCfg(appCfg),
		WithStore(st),
		WithLogger(l),
		WithMetrics(metrics),
		WithRaft(stubRaft),
		WithFutures(mockFutures),
	)

	require.IsType(t, &application{}, tapp)
	assert.NotNil(t, tapp.store)
	assert.NotNil(t, tapp.logger)
	assert.NotNil(t, tapp.cfg)
	assert.NotNil(t, tapp.metrics)
	assert.NotNil(t, tapp.raft)
	assert.NotNil(t, tapp.futures)
}
