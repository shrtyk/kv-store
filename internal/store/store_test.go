package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/snapshot"
	"github.com/shrtyk/kv-store/internal/tlog"
	tu "github.com/shrtyk/kv-store/tests/testutils"
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

func largeString(maxKeySize, maxValSize int) string {
	b := make([]byte, maxKeySize+maxValSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return string(hex.EncodeToString(b))
}
