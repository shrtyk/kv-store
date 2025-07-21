package store

import (
	"testing"

	"github.com/shrtyk/kv-store/internal/tlog"
	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	testFileName := tutils.FileWithCleanUp(t, "test")

	k := "test-key"
	v := "test-val"
	tl := tlog.MustCreateNewFileTransLog(testFileName)

	s := NewStore()
	tl.Start(t.Context(), s)

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

	assert.NoError(t, tl.Close())
}
