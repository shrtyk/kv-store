package tlog

import (
	"context"
	"sync"
	"testing"

	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTransactionFileLoggger(t *testing.T) {
	testFileName := tutils.FileWithCleanUp(t, "test")

	k, v := "test-key", "test-val"
	tl, err := NewFileTransactionalLogger(testFileName)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	tl.Start(ctx, nil)

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
