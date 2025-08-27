package testutils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
	"github.com/stretchr/testify/require"
)

func FileCleanUp(t *testing.T, filename string) {
	t.Helper()
	t.Cleanup(func() {
		if err := os.Remove(filename); err != nil {
			t.Errorf("failed to delete temporary test file: %v", err)
		}
	})
}

func NewMockLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func NewMockStoreCfg() *cfg.StoreCfg {
	return &cfg.StoreCfg{
		MaxKeySize: 100,
		MaxValSize: 100,

		// TryRebuildIn: 10 * time.Hour,
	}
}

func NewMockTransLogCfg() *cfg.WalCfg {
	return &cfg.WalCfg{
		LogFileName:        "test",
		MaxSizeBytes:       1048576,
		FsyncIn:            100 * time.Millisecond,
		FsyncRetriesAmount: 3,
		FsyncRetryIn:       1 * time.Millisecond,
	}
}

func NewMockSnapshotsCfg(dir string, maxSnapshots int) *cfg.SnapshotsCfg {
	return &cfg.SnapshotsCfg{
		SnapshotsDir:       dir,
		MaxSnapshotsAmount: maxSnapshots,
	}
}

func NewMockShardsCfg() *cfg.ShardsCfg {
	return &cfg.ShardsCfg{
		SparseRatio:        0.1,
		MinOpsUntilRebuild: 1000,
		MinDeletes:         500,
	}
}

func RandomString(t *testing.T, size int) string {
	t.Helper()
	b := make([]byte, size)
	_, err := rand.Read(b)
	require.NoError(t, err, "Failed to generate random bytes")
	return hex.EncodeToString(b)
}
