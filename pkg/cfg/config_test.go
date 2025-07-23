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
store_cfg:
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

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	testCases := []struct {
		name           string
		setup          func(t *testing.T)
		expectPanic    bool
		expectedEnv    string
		expectedMaxKey int
	}{
		{
			name: "success with flag",
			setup: func(t *testing.T) {
				os.Args = []string{"cmd", "-cfg_path=" + validConfigPath}
			},
			expectPanic:    false,
			expectedEnv:    "test",
			expectedMaxKey: 1024,
		},
		{
			name: "success with environment variable",
			setup: func(t *testing.T) {
				t.Setenv("CONFIG_PATH", validConfigPath)
			},
			expectPanic:    false,
			expectedEnv:    "test",
			expectedMaxKey: 1024,
		},
		{
			name: "flag over environment variable",
			setup: func(t *testing.T) {
				os.Args = []string{"cmd", "-cfg_path=" + validConfigPath}
				t.Setenv("CONFIG_PATH", "a/path/that/does/not/exist.yaml")
			},
			expectPanic:    false,
			expectedEnv:    "test",
			expectedMaxKey: 1024,
		},
		{
			name: "panic - config file does not exist",
			setup: func(t *testing.T) {
				os.Args = []string{"cmd", "-cfg_path=a/path/that/does/not/exist.yaml"}
			},
			expectPanic: true,
		},
		{
			name: "panic - invalid config file format",
			setup: func(t *testing.T) {
				os.Args = []string{"cmd", "-cfg_path=" + invalidConfigPath}
			},
			expectPanic: true,
		},
		{
			name: "panic - no config path provided",
			setup: func(t *testing.T) {
				os.Args = []string{"cmd"}
				t.Setenv("CONFIG_PATH", "")
			},
			expectPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			tc.setup(t)

			defer func() {
				r := recover()
				if tc.expectPanic {
					assert.NotNil(t, r)
				} else {
					assert.Nil(t, r)
				}
			}()

			cfg := ReadConfig()

			if !tc.expectPanic {
				assert.Equal(t, tc.expectedEnv, cfg.Env)
				assert.Equal(t, tc.expectedMaxKey, cfg.Store.MaxKeySize)
			}
		})
	}
}
