package main

import (
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/store"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	appCfg := &cfg.AppConfig{}
	l, _ := tu.NewMockLogger()
	s := store.NewStore(&sync.WaitGroup{}, &cfg.StoreCfg{}, &cfg.ShardsCfg{}, l)
	metrics := pmts.NewMockMetrics()

	tapp := NewApp()
	tapp.Init(
		WithCfg(appCfg),
		WithStore(s),
		WithLogger(l),
		WithMetrics(metrics),
	)

	require.IsType(t, &application{}, tapp)
	assert.NotNil(t, tapp.store)
	assert.NotNil(t, tapp.logger)
	assert.NotNil(t, tapp.cfg)
	assert.NotNil(t, tapp.metrics)
}
