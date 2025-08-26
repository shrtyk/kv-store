package main

import (
	"testing"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/shrtyk/kv-store/internal/core/snapshot"
	"github.com/shrtyk/kv-store/internal/core/store"
	"github.com/shrtyk/kv-store/internal/core/tlog"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	appCfg := &cfg.AppConfig{}
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)
	s := store.NewStore(&cfg.StoreCfg{}, l)
	metrics := pmts.NewMockMetrics()

	tapp := NewApp()
	tapp.Init(
		WithCfg(appCfg),
		WithStore(s),
		WithTransactionalLogger(tl),
		WithLogger(l),
		WithMetrics(metrics),
	)

	require.IsType(t, &application{}, tapp)
	assert.NotNil(t, tapp.store)
	assert.NotNil(t, tapp.logger)
	assert.NotNil(t, tapp.tl)
	assert.NotNil(t, tapp.cfg)
	assert.NotNil(t, tapp.metrics)
}
