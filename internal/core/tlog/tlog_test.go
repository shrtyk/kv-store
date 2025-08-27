package tlog

import (
	"context"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/core/snapshot"
	"github.com/shrtyk/kv-store/internal/core/store"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockstore struct {
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

	tl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	tl.WritePut(k, v)
	tl.WritePut(k, v)
	tl.WaitWritings()

	require.NoError(t, tl.Close())
	tl, err = NewFileTransactionalLogger(lcfg, l, snapshotter)
	require.NoError(t, err)

	events, errs := tl.ReadEvents()
	var count int
	for e := range events {
		count++
		assert.EqualValues(t, k, e.Key)
		assert.EqualValues(t, v, e.GetValue())
	}
	err = <-errs
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Test appending to existing log
	ntl := MustCreateNewFileTransLog(lcfg, l, snapshotter)
	defer func() {
		assert.NoError(t, ntl.Close())
	}()
	ntl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	ntl.WritePut(k, v)
	ntl.WritePut(k, v)
	ntl.WaitWritings()
	assert.EqualValues(t, uint64(4), ntl.lastSeq)
}

func TestSnapshotting(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	s := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)

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

type mockFile struct {
	syncErr error
	syncs   int
}

func (m *mockFile) Sync() error {
	m.syncs++
	return m.syncErr
}

func (m *mockFile) Stat() (os.FileInfo, error) {
	return nil, nil
}

func (m *mockFile) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockFile) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (m *mockFile) Close() error {
	return nil
}

func (m *mockFile) Name() string {
	return "mock"
}

func TestLastFsyncWithRetries(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	mockF := &mockFile{syncErr: assert.AnError}
	tl := &logger{
		cfg:  lcfg,
		log:  l,
		file: mockF,
	}

	tl.lastFsyncWithRetries()
	assert.Equal(t, 3, mockF.syncs)

	mockF.syncs = 0
	mockF.syncErr = nil
	tl.lastFsyncWithRetries()
	assert.Equal(t, 1, mockF.syncs)
}

type mockSnapshotter struct {
	snapshot.Snapshotter
	restoreErr     error
	restoreState   map[string]string
	findLatestPath string
	findLatestSeq  uint64
	findLatestErr  error
}

func (m *mockSnapshotter) Restore(path string) (map[string]string, error) {
	return m.restoreState, m.restoreErr
}

func (m *mockSnapshotter) FindLatest() (string, uint64, error) {
	return m.findLatestPath, m.findLatestSeq, m.findLatestErr
}

func TestRestore(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	s := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)
	snapshotter := &mockSnapshotter{
		findLatestPath: "test-path",
		findLatestSeq:  10,
		restoreState:   map[string]string{"key": "value"},
	}
	tl, err := NewFileTransactionalLogger(lcfg, l, snapshotter)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, tl.Close())
	}()
	tl.restore(s)
	val, err := s.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
	assert.Equal(t, uint64(10), tl.lastSeq)
}
