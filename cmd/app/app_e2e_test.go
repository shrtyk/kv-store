package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	rmocks "github.com/shrtyk/kv-store/internal/core/raft/mocks"
	"github.com/shrtyk/kv-store/internal/core/store"
	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	var wg sync.WaitGroup
	ap := NewApp()

	appCfg := &cfg.AppConfig{
		Env: "dev",
		Store: cfg.StoreCfg{
			MaxKeySize: 1024,
			MaxValSize: 1024,
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
		ShardsCfg: cfg.ShardsCfg{
			ShardsCount:        32,
			CheckFreq:          30 * time.Second,
			SparseRatio:        0.5,
			MinOpsUntilRebuild: 2000,
			MinDeletes:         500,
		},
	}
	l := logger.NewLogger(appCfg.Env)
	st := store.NewStore(&wg, &appCfg.Store, &appCfg.ShardsCfg, l)
	metric := pmts.NewMockMetrics()
	stubRaft := rmocks.NewStubRaft(st, true, 0)
	mockFutures := futuresmocks.NewMockFuturesStore(t)
	mockFuture := futuresmocks.NewMockFuture(t)

	mockFuture.On("Wait", mock.Anything).Return(nil)
	mockFutures.On("NewFuture", mock.Anything).Return(mockFuture)

	ap.Init(
		WithCfg(appCfg),
		WithLogger(l),
		WithStore(st),
		WithMetrics(metric),
		WithRaft(stubRaft),
		WithFutures(mockFutures),
		WithRaftPublicHTTPAddrs([]string{"http://localhost:16701"}),
	)

	router := ap.NewRouter()
	server := httptest.NewServer(router)
	defer server.Close()

	client := server.Client()
	addr := server.URL

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
	}, 1*time.Second, 10*time.Millisecond)

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
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
	require.NoError(t, delResp.Body.Close())

	// GET the key again to confirm deletion
	getReq, err = http.NewRequest(http.MethodGet, addr+"/v1/testkey", nil)
	assert.NoError(t, err)
	getResp, err = client.Do(getReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
	require.NoError(t, getResp.Body.Close())
}
