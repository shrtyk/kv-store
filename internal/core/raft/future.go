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
	pool     sync.Pool
}

func NewApplyFuture() *applyFuture {
	af := &applyFuture{
		promises: make(map[int64]*promise),
	}
	af.pool.New = func() any {
		return new(promise)
	}
	return af
}

func (af *applyFuture) StartGC(ctx context.Context) {
	go func() {
		// TODO: make configurable
		t := time.NewTicker(3 * time.Minute)
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
	af.mu.Lock()
	defer af.mu.Unlock()
	for i, p := range af.promises {
		if atomic.LoadUint32(&p.isStale) == stale || isClosed(p.done) {
			delete(af.promises, i)
			af.pool.Put(p)
		}
	}
}

func (af *applyFuture) NewFuture(logIdx int64) ftr.Future {
	af.mu.Lock()
	defer af.mu.Unlock()

	if p, exists := af.promises[logIdx]; exists && isClosed(p.done) {
		return p
	}

	p := af.pool.Get().(*promise)
	p.reset()
	af.promises[logIdx] = p
	return p
}

func isClosed(ch chan struct{}) bool {
	if ch == nil {
		return false
	}
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
		p := af.pool.Get().(*promise)
		p.reset()
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

func (p *promise) reset() {
	p.done = make(chan struct{})
	atomic.StoreUint32(&p.isStale, nonStale)
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
