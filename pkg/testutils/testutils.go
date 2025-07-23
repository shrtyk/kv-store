package tutils

import (
	"bytes"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/pkg/cfg"
)

func FileNameWithCleanUp(t *testing.T, filename string) string {
	t.Helper()
	t.Cleanup(func() {
		if err := os.Remove(filename); err != nil {
			t.Errorf("failed to delete temporary test file: %v", err)
		}
	})
	return filename
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
