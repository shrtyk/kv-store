package testutils

import (
	"bytes"
	"log/slog"

	"github.com/shrtyk/kv-store/internal/cfg"
)

func NewMockLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func NewMockStoreCfg() *cfg.StoreCfg {
	return &cfg.StoreCfg{
		MaxKeySize: 100,
		MaxValSize: 100,
	}
}

func NewMockShardsCfg() *cfg.ShardsCfg {
	return &cfg.ShardsCfg{
		SparseRatio:        0.1,
		MinOpsUntilRebuild: 1000,
		MinDeletes:         500,
	}
}
