package tlog

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/store"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockstore struct{
	store.Store
}

func (m *mockstore) Put(key, value string) error {
	return nil
}
func (m *mockstore) Get(key string) (string, error) {
	return "", nil
}
func (m *mockstore) Delete(key string) error {
	return nil
}

func (m *mockstore) Items() map[string]string {
	return nil
}

func TestTransactionFileLoggger(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	k, v := "test-key", "test-val"
	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl, err := NewFileTransactionalLogger(lcfg, l, snapshotter)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, tl.Close())
	}()

	tl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	tl.WritePut(k, v)
	tl.WritePut(k, v)

	events, errs := tl.ReadEvents()
	for e := range events {
		assert.EqualValues(t, k, e.key)
		assert.EqualValues(t, v, e.value)
	}
	assert.NoError(t, <-errs)
	tl.WaitWritings()

	ntl := MustCreateNewFileTransLog(lcfg, l, snapshotter)
	defer func() {
		assert.NoError(t, ntl.Close())
	}()
	ntl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	events, errs = ntl.ReadEvents()
	for range events {
	}
	assert.NoError(t, <-errs)

	ntl.WritePut(k, v)
	ntl.WritePut(k, v)
	ntl.WaitWritings()
	assert.EqualValues(t, uint64(4), ntl.lastSeq)
}

func TestTransactionLoggerCompacting(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	s := store.NewStore(tu.NewMockStoreCfg(), l)

	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl, err := NewFileTransactionalLogger(lcfg, l, snapshotter)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, tl.Close())
	}()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	tl.Start(ctx, &sync.WaitGroup{}, s)

	for i := range 10 {
		tl.WritePut(strconv.Itoa(i), strconv.Itoa(i+1))
	}

	for i := range 5 {
		tl.WriteDelete(strconv.Itoa(i))
	}

	tl.WaitWritings()

	tl.snapshot(s)
	tl.waitSnapshot()

	latestPath, latestSeq, err := snapshotter.FindLatest()
	require.NoError(t, err, "expected to find a snapshot")

	assert.Equal(t, uint64(15), latestSeq, "sequence number in snapshot should be the last known sequence number")

	restoredState, err := snapshotter.Restore(latestPath)
	require.NoError(t, err)

	assert.Len(t, restoredState, 5, "snapshot should contain 5 items")
	for i := 5; i < 10; i++ {
		key := strconv.Itoa(i)
		val, ok := restoredState[key]
		assert.True(t, ok, "expected key %s to be in snapshot", key)
		assert.Equal(t, strconv.Itoa(i+1), val, "incorrect value for key %s", key)
	}
}