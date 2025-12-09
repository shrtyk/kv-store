package raft

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ftr "github.com/shrtyk/kv-store/internal/core/ports/futures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyFuture_NewFuture_And_Fulfill(t *testing.T) {
	af := NewApplyFuture()
	logIdx := int64(1)

	future := af.NewFuture(logIdx)
	require.NotNil(t, future)

	var wg sync.WaitGroup
	wg.Add(1)
	var err error

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err = future.Wait(ctx)
	}()

	time.Sleep(10 * time.Millisecond)

	af.Fulfill(logIdx)
	wg.Wait()

	assert.NoError(t, err)
}

func TestApplyFuture_Fulfill_Before_NewFuture(t *testing.T) {
	af := NewApplyFuture()
	logIdx := int64(2)

	af.Fulfill(logIdx)

	future := af.NewFuture(logIdx)
	require.NotNil(t, future)

	err := future.Wait(context.Background())
	assert.NoError(t, err)
}

func TestFuture_Wait_Timeout(t *testing.T) {
	af := NewApplyFuture()
	logIdx := int64(3)

	future := af.NewFuture(logIdx)
	require.NotNil(t, future)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := future.Wait(ctx)
	assert.ErrorIs(t, err, ftr.ErrPromiseTimeout)

	af.mu.RLock()
	p, ok := af.promises[logIdx]
	af.mu.RUnlock()
	require.True(t, ok)
	assert.Equal(t, stale, atomic.LoadUint32(&p.isStale))
}

func TestApplyFuture_GC(t *testing.T) {
	af := NewApplyFuture()

	future1 := af.NewFuture(1)
	af.Fulfill(1)
	err := future1.Wait(context.Background())
	require.NoError(t, err)

	future2 := af.NewFuture(2)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = future2.Wait(ctx)
	require.ErrorIs(t, err, ftr.ErrPromiseTimeout)

	_ = af.NewFuture(3)

	af.mu.RLock()
	assert.Len(t, af.promises, 3)
	af.mu.RUnlock()

	af.cleanMap()

	af.mu.RLock()
	defer af.mu.RUnlock()
	assert.Len(t, af.promises, 1)
	_, exists := af.promises[3]
	assert.True(t, exists, "pending promise should remain")
	_, exists = af.promises[1]
	assert.False(t, exists, "fulfilled promise should be cleaned")
	_, exists = af.promises[2]
	assert.False(t, exists, "stale promise should be cleaned")
}

func TestIsClosed(t *testing.T) {
	t.Run("nil channel", func(t *testing.T) {
		assert.False(t, isClosed(nil))
	})

	t.Run("open channel", func(t *testing.T) {
		ch := make(chan struct{})
		assert.False(t, isClosed(ch))
	})

	t.Run("closed channel", func(t *testing.T) {
		ch := make(chan struct{})
		close(ch)
		assert.True(t, isClosed(ch))
	})
}

func TestApplyFuture_NewFuture_Idempotency(t *testing.T) {
	af := NewApplyFuture()
	logIdx := int64(5)

	future1 := af.NewFuture(logIdx)
	future2 := af.NewFuture(logIdx)

	assert.Same(t, future1, future2, "NewFuture should return the same promise for the same index")

	var wg sync.WaitGroup
	wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		err := future1.Wait(ctx)
		assert.NoError(t, err, "Wait on the first future should not time out")
	})

	time.Sleep(10 * time.Millisecond)
	af.Fulfill(logIdx)
	wg.Wait()
}
