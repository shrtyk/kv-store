package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	k := "test-key"
	v := "test-val"
	s := NewStore()

	_, err := s.Get(k)
	assert.ErrorIs(t, err, ErrorNoSuchKey)

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
	assert.ErrorIs(t, err, ErrorNoSuchKey)
}
