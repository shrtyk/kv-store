package logger

import (
	"testing"

	raftcorelog "github.com/shrtyk/raft-core/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestRaftLoggerCfg(t *testing.T) {
	testCases := []struct {
		name string
		env  string
		want raftcorelog.Enviroment
	}{
		{
			name: "dev environment",
			env:  "dev",
			want: raftcorelog.Dev,
		},
		{
			name: "prod environment",
			env:  "prod",
			want: raftcorelog.Prod,
		},
		{
			name: "empty: falls back to prod",
			env:  "",
			want: raftcorelog.Prod,
		},
		{
			name: "unknown: falls back to prod",
			env:  "staging",
			want: raftcorelog.Prod,
		},
		{
			name: "production environment is prod",
			env:  "production",
			want: raftcorelog.Prod,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := RaftLoggerCfg(tc.env)
			assert.Equal(t, tc.want, got)
		})
	}
}
