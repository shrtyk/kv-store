package tutils

import (
	"os"
	"testing"
)

func FileWithCleanUp(t *testing.T, filename string) string {
	t.Helper()
	t.Cleanup(func() {
		if err := os.Remove(filename); err != nil {
			t.Errorf("failed to delete temprorary test file: %v", err)
		}
	})
	return filename
}
