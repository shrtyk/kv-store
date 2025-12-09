package cfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadConfig_Env(t *testing.T) {
	originalPath := path
	path = ""
	defer func() { path = originalPath }()
	t.Setenv("CONFIG_PATH", "")

	t.Setenv("ENV", "test-env")
	t.Setenv("MAX_KEY_SIZE_BYTES", "123")
	t.Setenv("MAX_VAL_SIZE_BYTES", "456")
	t.Setenv("SHARDS_COUNT", "16")
	t.Setenv("HTTP_PORT", "9999")
	t.Setenv("GRPC_PORT", "9998")
	t.Setenv("RAFT_NODE_ID", "node-env")

	cfg := ReadConfig()

	assert.Equal(t, "test-env", cfg.Env)
	assert.Equal(t, 123, cfg.Store.MaxKeySize)
	assert.Equal(t, 456, cfg.Store.MaxValSize)
	assert.Equal(t, 16, cfg.ShardsCfg.ShardsCount)
	assert.Equal(t, "9999", cfg.HttpCfg.Port)
	assert.Equal(t, "9998", cfg.GRPCCfg.Port)
	assert.Equal(t, "node-env", cfg.Raft.NodeID)
}
