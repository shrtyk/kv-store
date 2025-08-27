package store

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
)

const (
	// Fallback number of shards
	DefaultShardsCount = 128
)

type Shard struct {
	cfg     *cfg.ShardsCfg
	mu      sync.RWMutex
	m       map[string]string
	puts    uint64
	deletes uint64
	maxSize int
}

type ShardedMap struct {
	shards     []*Shard
	hash       Hasher
	checksFreq time.Duration
}

func (s *Shard) needsRebuild() bool {
	s.mu.RLock()
	dels := s.deletes
	puts := s.puts
	maxSize := s.maxSize
	curSize := len(s.m)
	s.mu.RUnlock()

	totalOps := puts + dels
	if dels >= uint64(s.cfg.MinDeletes) &&
		curSize <= int(float64(maxSize)*s.cfg.SparseRatio) &&
		totalOps >= uint64(s.cfg.MinOpsUntilRebuild) {
		return true
	}

	return false
}

func (s *Shard) rebuild() {
	s.mu.RLock()
	newMap := make(map[string]string, len(s.m))
	maps.Copy(newMap, s.m)
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.m = newMap
	s.puts = 0
	s.deletes = 0
	s.maxSize = len(s.m)
}

type Hasher interface {
	Sum64(string) uint64
}

func NewShardedMap(shardsCfg *cfg.ShardsCfg, shardsCount int, hasher Hasher) *ShardedMap {
	if shardsCount <= 0 {
		shardsCount = DefaultShardsCount
	}

	shards := make([]*Shard, shardsCount)
	for i := 0; i < shardsCount; i++ {
		shards[i] = &Shard{
			cfg: shardsCfg,
			m:   make(map[string]string),
		}
	}

	return &ShardedMap{
		shards:     shards,
		hash:       hasher,
		checksFreq: shardsCfg.CheckFreq,
	}
}

func (m *ShardedMap) getShard(key string) *Shard {
	return m.shards[m.hash.Sum64(key)%uint64(len(m.shards))]
}

func (m *ShardedMap) Put(key, value string) {
	shard := m.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.m[key] = value
	shard.puts++
	shard.maxSize = max(shard.maxSize, len(shard.m))
}

func (m *ShardedMap) Get(key string) (string, bool) {
	shard := m.getShard(key)

	shard.mu.RLock()
	defer shard.mu.RUnlock()

	val, ok := shard.m[key]
	return val, ok
}

func (m *ShardedMap) Delete(key string) {
	shard := m.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.m, key)
	shard.deletes++
}

func (m *ShardedMap) Len() int {
	count := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		count += len(shard.m)
		shard.mu.RUnlock()
	}
	return count
}

func (m *ShardedMap) Items() map[string]string {
	items := make(map[string]string)
	for _, shard := range m.shards {
		shard.mu.RLock()
		maps.Copy(items, shard.m)
		shard.mu.RUnlock()
	}
	return items
}

func (m *ShardedMap) StartShardsSupervisor(ctx context.Context, wg *sync.WaitGroup) {
	t := time.NewTicker(m.checksFreq)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				for _, shard := range m.shards {
					if shard.needsRebuild() {
						shard.rebuild()
					}
				}
			}
		}
	}()
}
