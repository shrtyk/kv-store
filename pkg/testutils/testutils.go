package tutils

import (
	"bytes"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/pkg/cfg"
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

		TryRebuildIn: 10 * time.Hour,
	}
}

func NewMockTransLogCfg() *cfg.TransLoggerCfg {
	return &cfg.TransLoggerCfg{
		LogFileName: "test",
		FsyncIn:     100 * time.Millisecond,
	}
}
