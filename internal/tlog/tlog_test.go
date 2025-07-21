package tlog

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionFileLoggger(t *testing.T) {
	k, v := "test-key", "test-val"
	l, err := NewFileTransactionalLogger("test")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	l.Start(ctx)
	l.WritePut(k, v)
	l.WriteDelete(k)

	events, errs := l.ReadEvents()
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
}
