package cfg

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	tempDir := t.TempDir()

	validConfigContent := `
env: "test"
store:
  max_key: 1024
  max_val: 4096
`
	validConfigPath := filepath.Join(tempDir, "valid.yaml")
	err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644)
	require.NoError(t, err)

	invalidConfigContent := `wrong yaml`
	invalidConfigPath := filepath.Join(tempDir, "wrong.yaml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfigContent), 0644)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		setup          func(t *testing.T)
		cleanup        func(t *testing.T)
		expectedEnv    string
		expectedMaxKey int
		expectedMaxVal int
	}{
		{
			name: "success with flag",
			setup: func(t *testing.T) {
				err := flag.Set("cfg_path", validConfigPath)
				require.NoError(t, err)
			},
			cleanup: func(t *testing.T) {
				err := flag.Set("cfg_path", "")
				require.NoError(t, err)
				path = ""
			},
			expectedEnv:    "test",
			expectedMaxKey: 1024,
			expectedMaxVal: 4096,
		},
		{
			name: "success with environment variable",
			setup: func(t *testing.T) {
				t.Setenv("CONFIG_PATH", validConfigPath)
			},
			cleanup:        func(t *testing.T) {},
			expectedEnv:    "test",
			expectedMaxKey: 1024,
			expectedMaxVal: 4096,
		},
		{
			name: "fallback to env when config file is invalid",
			setup: func(t *testing.T) {
				t.Setenv("CONFIG_PATH", invalidConfigPath)
				t.Setenv("ENV", "env-fallback-invalid")
			},
			cleanup:        func(t *testing.T) {},
			expectedEnv:    "env-fallback-invalid",
			expectedMaxKey: 1024, // default
			expectedMaxVal: 1024, // default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)
			t.Cleanup(func() { tc.cleanup(t) })

			cfg := ReadConfig()

			assert.Equal(t, tc.expectedEnv, cfg.Env)
			assert.Equal(t, tc.expectedMaxKey, cfg.Store.MaxKeySize)
			assert.Equal(t, tc.expectedMaxVal, cfg.Store.MaxValSize)
		})
	}
}
