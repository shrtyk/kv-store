package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	metrics "github.com/shrtyk/kv-store/pkg/prometheus"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)
	defer func() {
		assert.NoError(t, tl.Close())
	}()

	store := store.NewStore(tu.NewMockStoreCfg(), l)
	m := metrics.NewMockMetrics()

	app := NewApp()
	app.Init(
		WithCfg(&cfg.AppConfig{}),
		WithLogger(l),
		WithMetrics(m),
		WithTransactionalLogger(tl),
		WithStore(store),
	)

	router := app.NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	b, err := io.ReadAll(w.Body)
	assert.NoError(t, err)
	assert.Equal(t, "404 page not found\n", string(b))
}
