package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSnapshotter(t *testing.T) {
	l, _ := tu.NewMockLogger()
	tempDir := t.TempDir()
	snapshotter := NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(tempDir, 2),
		l,
	)

	// No snapshots initially
	_, _, err := snapshotter.FindLatest()
	assert.ErrorIs(t, err, os.ErrNotExist)

	// Create a snapshot
	state1 := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	var seq1 uint64 = 10
	snapshotPath1, err := snapshotter.Create(state1, seq1)
	require.NoError(t, err)
	assert.FileExists(t, snapshotPath1)

	// Restore from the snapshot
	restoredState1, err := snapshotter.Restore(snapshotPath1)
	require.NoError(t, err)
	assert.Equal(t, state1, restoredState1)

	// Find the latest snapshot
	latestPath, latestSeq, err := snapshotter.FindLatest()
	require.NoError(t, err)
	assert.Equal(t, snapshotPath1, latestPath)
	assert.Equal(t, seq1, latestSeq)

	// Create a second, later snapshot
	state2 := map[string]string{
		"key3": "value3",
	}
	var seq2 uint64 = 20
	snapshotPath2, err := snapshotter.Create(state2, seq2)
	require.NoError(t, err)

	// Find the new latest snapshot
	latestPath, latestSeq, err = snapshotter.FindLatest()
	require.NoError(t, err)
	assert.Equal(t, snapshotPath2, latestPath)
	assert.Equal(t, seq2, latestSeq)

	// Restore from the second snapshot
	restoredState2, err := snapshotter.Restore(snapshotPath2)
	require.NoError(t, err)
	assert.Equal(t, state2, restoredState2)
}

func TestRestoreWithMalformedData(t *testing.T) {
	l, _ := tu.NewMockLogger()
	tempDir := t.TempDir()
	snapshotter := NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(tempDir, 2),
		l,
	)

	// Create a snapshot file with some malformed lines
	snapshotPath := filepath.Join(tempDir, "malformed.dat")
	content := `key1	value1
key2	value2
malformed_line

key3	value3`
	err := os.WriteFile(snapshotPath, []byte(content), 0644)
	require.NoError(t, err)

	expectedState := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	restoredState, err := snapshotter.Restore(snapshotPath)
	require.NoError(t, err)
	assert.Equal(t, expectedState, restoredState)
}

func TestFindLatest_MultipleSnapshots(t *testing.T) {
	l, _ := tu.NewMockLogger()
	tempDir := t.TempDir()
	snapshotter := NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(tempDir, 2),
		l,
	)

	_, err := snapshotter.Create(map[string]string{"key1": "value1"}, 10)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = snapshotter.Create(map[string]string{"key2": "value2"}, 30)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	// Third snapshot (latest in time, but lower sequence number than second one)
	state3 := map[string]string{"key3": "value3"}
	path3, err := snapshotter.Create(state3, 20)
	require.NoError(t, err)

	// Find the latest snapshot
	latestPath, latestSeq, err := snapshotter.FindLatest()
	require.NoError(t, err)

	assert.Equal(t, path3, latestPath)
	assert.Equal(t, uint64(20), latestSeq)
}

func TestSnapshotClean(t *testing.T) {
	l, _ := tu.NewMockLogger()
	tempDir := t.TempDir()
	snapshotter := NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(tempDir, 2),
		l,
	)

	// snapshot 1
	snap1Path, err := snapshotter.Create(map[string]string{"key1": "value1"}, 10)
	require.NoError(t, err)
	assert.FileExists(t, snap1Path)

	time.Sleep(10 * time.Millisecond)

	// snapshot 2
	snap2Path, err := snapshotter.Create(map[string]string{"key2": "value2"}, 20)
	require.NoError(t, err)
	assert.FileExists(t, snap2Path)

	files, _ := os.ReadDir(tempDir)
	assert.Len(t, files, 2)

	time.Sleep(10 * time.Millisecond)

	// snapshot 3, which should trigger the rotation
	snap3Path, err := snapshotter.Create(map[string]string{"key3": "value3"}, 30)
	require.NoError(t, err)
	assert.FileExists(t, snap3Path)

	files, _ = os.ReadDir(tempDir)
	assert.Len(t, files, 2, "expected only 2 snapshots after rotation")

	_, err = os.Stat(snap1Path)
	assert.True(t, os.IsNotExist(err), "expected the oldest snapshot to be deleted")

	assert.FileExists(t, snap2Path)
	assert.FileExists(t, snap3Path)
}
