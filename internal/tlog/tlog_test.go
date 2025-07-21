package tlog

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionFileLoggger(t *testing.T) {
	testFileName := "test"
	t.Cleanup(func() {
		if err := os.Remove(testFileName); err != nil {
			t.Errorf("failed to delete temprorary test file: %v", err)
		}
	})

	k, v := "test-key", "test-val"
	tl, err := NewFileTransactionalLogger(testFileName)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	tl.Start(ctx)

	tl.WritePut(k, v)
	tl.WriteDelete(k)

	events, errs := tl.ReadEvents()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for e := range events {
			assert.EqualValues(t, k, e.key)
			assert.EqualValues(t, v, e.value)
		}
	}()
	go func() {
		defer wg.Done()
		for err := range errs {
			assert.NoError(t, err)
		}
	}()

	cancel()
	wg.Wait()

	assert.NoError(t, tl.Close())
}

func FileWithCleanUp(t *testing.T, filename string) string {
	t.Helper()
	t.Cleanup(func() {
		if err := os.Remove(filename); err != nil {
			t.Errorf("failed to delete temprorary test file: %v", err)
		}
	})
	return filename
}
