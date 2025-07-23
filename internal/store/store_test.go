package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"

	"github.com/shrtyk/kv-store/internal/tlog"
	"github.com/shrtyk/kv-store/pkg/cfg"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	l, _ := tu.NewMockLogger()
	testFileName := tu.FileNameWithCleanUp(t, "test")

	k := "test-key"
	v := "test-val"
	tl := tlog.MustCreateNewFileTransLog(testFileName, l)

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

	lString := largeString(s.cfg.MaxKeySize, s.cfg.MaxValKey)
	err = s.Put(lString, "val")
	assert.ErrorIs(t, err, ErrKeyTooLarge)
	err = s.Put("key", lString)
	assert.ErrorIs(t, err, ErrValueTooLarge)

	assert.NoError(t, tl.Close())
}

func TestLargeKeyAndVal(t *testing.T) {
	l, _ := tu.NewMockLogger()
	testFileName := tu.FileNameWithCleanUp(t, "test")
	tl := tlog.MustCreateNewFileTransLog(testFileName, l)

	s := NewStore(&cfg.StoreCfg{}, l)
	tl.Start(t.Context(), &sync.WaitGroup{}, s)

}

func largeString(maxKeySize, maxValSize int) string {
	b := make([]byte, maxKeySize+maxValSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return string(hex.EncodeToString(b))
}
