package app_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/app"
	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	"github.com/shrtyk/kv-store/pkg/logger"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
	tutils "github.com/shrtyk/kv-store/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	ap := app.NewApp()
	tempDir := t.TempDir()
	logName := "wal.log"
	tutils.FileCleanUp(t, logName)

	cfg := &cfg.AppConfig{
		Env: "dev",
		Store: cfg.StoreCfg{
			MaxKeySize:  1024,
			MaxValSize:  1024,
			ShardsCount: 32,
		},
		Wal: cfg.WalCfg{
			LogFileName:        logName,
			MaxSizeBytes:       10485760,
			FsyncIn:            300 * time.Millisecond,
			FsyncRetriesAmount: 3,
			FsyncRetryIn:       500 * time.Millisecond,
		},
		Snapshots: cfg.SnapshotsCfg{
			SnapshotsDir:       tempDir,
			MaxSnapshotsAmount: 2,
		},
		HttpCfg: cfg.HttpCfg{
			Host:               "localhost",
			Port:               "16701",
			ServerIdleTimeout:  5 * time.Second,
			ServerWriteTimeout: 10 * time.Second,
			ServerReadTimeout:  10 * time.Second,
		},
		GRPCCfg: cfg.GRPCCfg{
			Port: "16702",
		},
	}
	l := logger.NewLogger(cfg.Env)
	st := store.NewStore(&cfg.Store, l)
	snapshotter := snapshot.NewFileSnapshotter(&cfg.Snapshots, l)
	tl := tlog.MustCreateNewFileTransLog(&cfg.Wal, l, snapshotter)
	metric := metrics.NewMockMetrics()

	ap.Init(
		app.WithCfg(cfg),
		app.WithLogger(l),
		app.WithStore(st),
		app.WithTransactionalLogger(tl),
		app.WithMetrics(metric),
	)

	go func() {
		ap.Serve()
	}()

	client := &http.Client{}
	addr := fmt.Sprintf("http://%s:%s", cfg.HttpCfg.Host, cfg.HttpCfg.Port)

	// wait for the server to start
	require.Eventually(t, func() bool {
		req, err := http.NewRequest(http.MethodGet, addr+"/healthz", nil)
		if err != nil {
			return false
		}
		resp, err := client.Do(req)
		if err != nil {
			return false
		}
		defer require.NoError(t, resp.Body.Close())
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond)

	// PUT a key-value pair
	putReq, err := http.NewRequest(http.MethodPut, addr+"/v1/testkey", strings.NewReader("testvalue"))
	assert.NoError(t, err)
	putResp, err := client.Do(putReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, putResp.StatusCode)
	require.NoError(t, putResp.Body.Close())

	// GET the value
	getReq, err := http.NewRequest(http.MethodGet, addr+"/v1/testkey", nil)
	assert.NoError(t, err)
	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	body, err := io.ReadAll(getResp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "testvalue", string(body))
	require.NoError(t, getResp.Body.Close())

	// DELETE the key
	delReq, err := http.NewRequest(http.MethodDelete, addr+"/v1/testkey", nil)
	assert.NoError(t, err)
	delResp, err := client.Do(delReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
	require.NoError(t, delResp.Body.Close())

	// GET the key again to confirm deletion
	getReq, err = http.NewRequest(http.MethodGet, addr+"/v1/testkey", nil)
	assert.NoError(t, err)
	getResp, err = client.Do(getReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	require.NoError(t, getResp.Body.Close())
}
