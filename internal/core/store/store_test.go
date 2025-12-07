package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"

	pstore "github.com/shrtyk/kv-store/internal/core/ports/store"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	l, _ := tu.NewMockLogger()

	k := "test-key"
	v := "test-val"

	s := NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)

	_, err := s.Get(k)
	assert.ErrorIs(t, err, pstore.ErrNoSuchKey)

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
	assert.ErrorIs(t, err, pstore.ErrNoSuchKey)

	lString := largeString(s.cfg.MaxKeySize, s.cfg.MaxValSize)
	err = s.Put(lString, "val")
	assert.ErrorIs(t, err, pstore.ErrKeyTooLarge)
	err = s.Put("key", lString)
	assert.ErrorIs(t, err, pstore.ErrValueTooLarge)
}

func largeString(maxKeySize, maxValSize int) string {
	b := make([]byte, maxKeySize+maxValSize)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return string(hex.EncodeToString(b))
}
