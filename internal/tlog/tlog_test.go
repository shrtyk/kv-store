package tlog

import (
	"context"
	"testing"

	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

type mockstore struct{}

func (m *mockstore) Put(key, value string) error {
	return nil
}
func (m *mockstore) Get(key string) (string, error) {
	return "", nil
}
func (m *mockstore) Delete(key string) error {
	return nil
}

func TestTransactionFileLoggger(t *testing.T) {
	testFileName := tutils.FileNameWithCleanUp(t, "test")

	k, v := "test-key", "test-val"
	tl, err := NewFileTransactionalLogger(testFileName)
	assert.NoError(t, err)
	defer tl.Close()

	tl.Start(context.Background(), &mockstore{})

	tl.WritePut(k, v)
	tl.WritePut(k, v)

	events, errs := tl.ReadEvents()
	for e := range events {
		assert.EqualValues(t, k, e.key)
		assert.EqualValues(t, v, e.value)
	}
	assert.NoError(t, <-errs)
	tl.Wait()

	ntl := MustCreateNewFileTransLog(testFileName)
	defer ntl.Close()
	ntl.Start(context.Background(), &mockstore{})

	events, errs = ntl.ReadEvents()
	for range events {
	}
	assert.NoError(t, <-errs)

	ntl.WritePut(k, v)
	ntl.WritePut(k, v)
	ntl.Wait()
	assert.EqualValues(t, 4, ntl.lastSeq)
}
