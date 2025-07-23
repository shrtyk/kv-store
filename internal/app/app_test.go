package app

import (
	"testing"

	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	l, _ := tutils.NewMockLogger()
	logfile := tutils.FileNameWithCleanUp(t, "test")
	tl := tlog.MustCreateNewFileTransLog(logfile, l)
	s := store.NewStore(&cfg.StoreCfg{}, l)

	tapp := NewApp()
	tapp.Init(
		WithStore(s),
		WithTransactionalLogger(tl),
		WithLogger(l),
	)

	require.IsType(t, &application{}, tapp)
	assert.NotNil(t, tapp.store)
	assert.NotNil(t, tapp.logger)
	assert.NotNil(t, tapp.tl)
}
