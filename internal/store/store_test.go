package store

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/shrtyk/kv-store/internal/tlog"
	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	testFileName := tutils.FileNameWithCleanUp(t, "test")

	k := "test-key"
	v := "test-val"
	tl := tlog.MustCreateNewFileTransLog(testFileName)

	s := NewStore()
	tl.Start(t.Context(), s)

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

	err = s.Put(largeString(), "val")
	assert.ErrorIs(t, err, ErrKeyTooLarge)
	err = s.Put("key", largeString())
	assert.ErrorIs(t, err, ErrValueTooLarge)

	assert.NoError(t, tl.Close())
}

func TestLargeKeyAndVal(t *testing.T) {
	testFileName := tutils.FileNameWithCleanUp(t, "test")
	tl := tlog.MustCreateNewFileTransLog(testFileName)

	s := NewStore()
	tl.Start(t.Context(), s)

}

func largeString() string {
	b := make([]byte, maxKeySize+maxValSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return string(hex.EncodeToString(b))
}
