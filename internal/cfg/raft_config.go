package cfg

import (
	"fmt"
	"strings"
	"time"

	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/shrtyk/raft-core/api"
)

type RaftCfg struct {
	NodeID                     string        `yaml:"node_id" env:"RAFT_NODE_ID"`
	Peers                      []string      `yaml:"peers" env:"RAFT_PEERS" env-separator:","`
	PublicHTTPAddrs            []string      `yaml:"public_http_addrs" env:"RAFT_PUBLIC_HTTP_ADDRS" env-separator:","`
	DataDir                    string        `yaml:"data_dir" env:"RAFT_DATA_DIR" env-default:"./data/raft"`
	CommitNoOpOn               bool          `yaml:"commit_noop_on_start" env:"RAFT_COMMIT_NOOP_ON_START" env-default:"true"`
	GRPCAddr                   string        `yaml:"grpc_addr" env:"RAFT_GRPC_ADDR"`
	MonitoringAddr             string        `yaml:"monitoring_addr" env:"RAFT_MONITORING_ADDR"`
	ElectionTimeout            time.Duration `yaml:"election_timeout" env:"RAFT_ELECTION_TIMEOUT" env-default:"150ms"`
	ElectionTimeoutRandomDelta time.Duration `yaml:"election_timeout_random_delta" env:"RAFT_ELECTION_TIMEOUT_RANDOM_DELTA" env-default:"150ms"`
	HeartbeatTimeout           time.Duration `yaml:"heartbeat_timeout" env:"RAFT_HEARTBEAT_TIMEOUT" env-default:"100ms"`
	RPCTimeout                 time.Duration `yaml:"rpc_timeout" env:"RAFT_RPC_TIMEOUT" env-default:"100ms"`
	ShutdownTimeout            time.Duration `yaml:"shutdown_timeout" env:"RAFT_SHUTDOWN_TIMEOUT" env-default:"5s"`
	SnapshotThreshold          int           `yaml:"snapshot_threshold_bytes" env:"RAFT_SNAPSHOT_THRESHOLD_BYTES" env-default:"8388608"`
	SnapshotCheckIn            time.Duration `yaml:"snapshot_check_in" env:"RAFT_SNAPSHOT_CHECK_IN" env-default:"1s"`
	LinearizableReadIn         time.Duration `yaml:"linearizable_read_in" env:"RAFT_LINEARIZABLE_READ_IN" env-default:"100ms"`
	CBFailureThreshold         int           `yaml:"cb_failure_threshold" env:"RAFT_CB_FAILURE_THRESHOLD" env-default:"8"`
	CBSuccessThreshold         int           `yaml:"cb_success_threshold" env:"RAFT_CB_SUCCESS_THRESHOLD" env-default:"4"`
	CBResetTimeout             time.Duration `yaml:"cb_reset_timeout" env:"CB_RESET_TIMEOUT" env-default:"5s"`
}

func (r *RaftCfg) MapToRaftApiCfg(env string) *api.RaftConfig {
	return &api.RaftConfig{
		Log: api.LoggerCfg{
			Env: logger.RaftLoggerCfg(env),
		},
		Timings: api.RaftTimings{
			ElectionTimeoutBase:        r.ElectionTimeout,
			ElectionTimeoutRandomDelta: r.ElectionTimeoutRandomDelta,
			HeartbeatTimeout:           r.HeartbeatTimeout,
			RPCTimeout:                 r.RPCTimeout,
			ShutdownTimeout:            r.ShutdownTimeout,
		},
		CBreaker: api.CircuitBreakerCfg{
			FailureThreshold: r.CBFailureThreshold,
			SuccessThreshold: r.CBSuccessThreshold,
			ResetTimeout:     r.CBResetTimeout,
		},
		Snapshots: api.SnapshotsCfg{
			CheckLogSizeInterval: r.SnapshotCheckIn,
			ThresholdBytes:       r.SnapshotThreshold,
		},
		HttpMonitoringAddr: r.MonitoringAddr,
		GRPCAddr:           r.GRPCAddr,
		CommitNoOpOn:       r.CommitNoOpOn,
	}
}

type ParsedPeers struct {
	Me    int
	Addrs []string
}

func (r *RaftCfg) ParsePeers() (*ParsedPeers, error) {
	var me int
	var found bool
	addrs := make([]string, len(r.Peers))
	for i, peer := range r.Peers {
		parts := strings.Split(peer, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid peer format: %s", peer)
		}
		if parts[0] == r.NodeID {
			me = i
			found = true
		}
		addrs[i] = fmt.Sprintf("%s:%s", parts[1], parts[2])
	}
	if !found {
		return nil, fmt.Errorf("node id %s not found in peers list", r.NodeID)
	}

	return &ParsedPeers{
		Me:    me,
		Addrs: addrs,
	}, nil
}
