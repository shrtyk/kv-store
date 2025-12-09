package cfg

import (
	"testing"
	"time"

	"github.com/shrtyk/kv-store/pkg/logger"
	raftapi "github.com/shrtyk/raft-core/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRaftCfg_MapToRaftApiCfg(t *testing.T) {
	rcfg := &RaftCfg{
		NodeID:                     "node-1",
		Peers:                      []string{"node-1:kv-store-1:16801", "node-2:kv-store-2:16802"},
		PublicHTTPAddrs:            []string{"http://localhost:8081", "http://localhost:8082"},
		DataDir:                    "/tmp/raft",
		CommitNoOpOn:               true,
		GRPCAddr:                   "localhost:9090",
		MonitoringAddr:             "localhost:9091",
		ElectionTimeout:            1 * time.Second,
		ElectionTimeoutRandomDelta: 2 * time.Second,
		HeartbeatTimeout:           3 * time.Second,
		RPCTimeout:                 4 * time.Second,
		ShutdownTimeout:            5 * time.Second,
		SnapshotThreshold:          1024,
		SnapshotCheckIn:            6 * time.Second,
		LinearizableReadIn:         7 * time.Second,
		CBFailureThreshold:         8,
		CBSuccessThreshold:         9,
		CBResetTimeout:             10 * time.Second,
		BatchSize:                  11,
		FsyncTimeout:               12 * time.Second,
	}

	env := "dev"
	apiCfg := rcfg.MapToRaftApiCfg(env)

	require.NotNil(t, apiCfg)

	expectedAPICfg := &raftapi.RaftConfig{
		Log: raftapi.LoggerCfg{
			Env: logger.RaftLoggerCfg(env),
		},
		Timings: raftapi.RaftTimings{
			ElectionTimeoutBase:        1 * time.Second,
			ElectionTimeoutRandomDelta: 2 * time.Second,
			HeartbeatTimeout:           3 * time.Second,
			RPCTimeout:                 4 * time.Second,
			ShutdownTimeout:            5 * time.Second,
		},
		CBreaker: raftapi.CircuitBreakerCfg{
			FailureThreshold: 8,
			SuccessThreshold: 9,
			ResetTimeout:     10 * time.Second,
		},
		Fsync: raftapi.FsyncCfg{
			BatchSize: 11,
			Timeout:   12 * time.Second,
		},
		Snapshots: raftapi.SnapshotsCfg{
			CheckLogSizeInterval: 6 * time.Second,
			ThresholdBytes:       1024,
		},
		HttpMonitoringAddr: "localhost:9091",
		GRPCAddr:           "localhost:9090",
		CommitNoOpOn:       true,
	}

	assert.Equal(t, expectedAPICfg.Log, apiCfg.Log)
	assert.Equal(t, expectedAPICfg.Timings, apiCfg.Timings)
	assert.Equal(t, expectedAPICfg.CBreaker, apiCfg.CBreaker)
	assert.Equal(t, expectedAPICfg.Fsync, apiCfg.Fsync)
	assert.Equal(t, expectedAPICfg.Snapshots, apiCfg.Snapshots)
	assert.Equal(t, expectedAPICfg.HttpMonitoringAddr, apiCfg.HttpMonitoringAddr)
	assert.Equal(t, expectedAPICfg.GRPCAddr, apiCfg.GRPCAddr)
	assert.Equal(t, expectedAPICfg.CommitNoOpOn, apiCfg.CommitNoOpOn)
}

func TestRaftCfg_ParsePeers(t *testing.T) {
	testCases := []struct {
		name       string
		rcfg       *RaftCfg
		want       *ParsedPeers
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success",
			rcfg: &RaftCfg{
				NodeID: "node-2",
				Peers:  []string{"node-1:kv-store-1:16801", "node-2:kv-store-2:16802", "node-3:kv-store-3:16803"},
			},
			want: &ParsedPeers{
				Me:    1,
				Addrs: []string{"kv-store-1:16801", "kv-store-2:16802", "kv-store-3:16803"},
			},
			wantErr: false,
		},
		{
			name: "invalid peer format",
			rcfg: &RaftCfg{
				NodeID: "node-1",
				Peers:  []string{"node-1:kv-store-1:16801", "invalid-peer"},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "invalid peer format: invalid-peer",
		},
		{
			name: "node id not found",
			rcfg: &RaftCfg{
				NodeID: "node-4",
				Peers:  []string{"node-1:kv-store-1:16801", "node-2:kv-store-2:16802", "node-3:kv-store-3:16803"},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "node id node-4 not found in peers list",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := tc.rcfg.ParsePeers()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				assert.Nil(t, parsed)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, parsed)
			}
		})
	}
}
