package tutils

import (
	"os"
	"testing"
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
