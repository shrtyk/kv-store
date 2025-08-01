package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	k := "test-key"
	v := "test-val"
	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)

	s := NewStore(tu.NewMockStoreCfg(), l)
	tl.Start(t.Context(), &sync.WaitGroup{}, s)

	_, err := s.Get(k)
	assert.ErrorIs(t, err, ErrNoSuchKey)

	err = s.Put(k, v)
	assert.NoError(t, err)

	val, err := s.Get(k)
	assert.NoError(t, err)
	assert.EqualValuesf(t, "test-val", val, "expected: %s, got: %s", v, val)

	err = s.Delete("wrong-key")
	assert.NoError(t, err)
	val, _ = s.Get(k)
	assert.EqualValuesf(t, v, val, "expected: %s, got: %s", v, val)

	err = s.Delete(k)
	assert.NoError(t, err)
	_, err = s.Get(k)
	assert.ErrorIs(t, err, ErrNoSuchKey)

	lString := largeString(s.cfg.MaxKeySize, s.cfg.MaxValSize)
	err = s.Put(lString, "val")
	assert.ErrorIs(t, err, ErrKeyTooLarge)
	err = s.Put("key", lString)
	assert.ErrorIs(t, err, ErrValueTooLarge)

	assert.NoError(t, tl.Close())
}

func TestBackgroundMapRebuilder(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	snapshotter := snapshot.NewFileSnapshotter(
		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
		l,
	)
	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)

	s := NewStore(&cfg.StoreCfg{
		MaxKeySize:          100,
		MaxValSize:          100,
		TryRebuildIn:        20 * time.Millisecond,
		MinDeletesTrigger:   5,
		SparseRatio:         0.5,
		MinOpsBeforeRebuild: 10,
	}, l)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var wg sync.WaitGroup
	tl.Start(ctx, &wg, s)
	s.StartMapRebuilder(ctx, &wg)

	for i := range 10 {
		s.Put(strconv.Itoa(i), strconv.Itoa(i))
	}

	for i := range 6 {
		s.Delete(strconv.Itoa(i))
	}

	assert.Eventually(t, func() bool {
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.puts == 0 && s.deletions == 0
	},
		500*time.Millisecond,
		10*time.Millisecond,
		"rebuilder didn't reset puts and deteions int time")

	s.mu.RLock()
	assert.EqualValues(t, 4, s.maxSize)
	assert.EqualValues(t, 0, s.puts)
	assert.EqualValues(t, 0, s.deletions)
	s.mu.RUnlock()

	cancel()
}

func largeString(maxKeySize, maxValSize int) string {
	b := make([]byte, maxKeySize+maxValSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return string(hex.EncodeToString(b))
}
