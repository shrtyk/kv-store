package raft

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	ftr "github.com/shrtyk/kv-store/internal/core/ports/futures"
)

var (
	_ ftr.FuturesStore = (*applyFuture)(nil)
	_ ftr.Future       = (*promise)(nil)
)

type applyFuture struct {
	mu       sync.RWMutex
	promises map[int64]*promise
}

func NewApplyFuture() *applyFuture {
	return &applyFuture{
		promises: make(map[int64]*promise),
	}
}

func (af *applyFuture) StartGC(ctx context.Context) {
	go func() {
		// TODO: make configurable
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				af.cleanMap()
			}
		}
	}()
}

func (af *applyFuture) cleanMap() {
	af.mu.RLock()
	newMap := make(map[int64]*promise, len(af.promises))
	for i, p := range af.promises {
		if !(atomic.LoadUint32(&p.isStale) == stale) {
			newMap[i] = p
		}
	}
	af.mu.RUnlock()

	af.mu.Lock()
	defer af.mu.Unlock()
	af.promises = newMap
}

func (af *applyFuture) NewPromise(logIdx int64) ftr.Future {
	af.mu.Lock()
	defer af.mu.Unlock()

	if p, exists := af.promises[logIdx]; exists && isClosed(p.done) {
		return p
	}

	p := NewPromise()
	af.promises[logIdx] = p
	return p
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func (af *applyFuture) Fulfill(logIdx int64) {
	af.mu.Lock()
	defer af.mu.Unlock()
	if p, exists := af.promises[logIdx]; exists {
		close(p.done)
	} else {
		p := NewPromise()
		close(p.done)
		af.promises[logIdx] = p
	}
}

const (
	nonStale uint32 = iota
	stale
)

type promise struct {
	isStale uint32
	done    chan struct{}
}

func NewPromise() *promise {
	return &promise{
		isStale: nonStale,
		done:    make(chan struct{}),
	}
}

func (p *promise) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		atomic.StoreUint32(&p.isStale, stale)
		return ftr.ErrPromiseTimeout
	case <-p.done:
		return nil
	}
}
