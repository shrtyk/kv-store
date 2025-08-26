package store

import (
	"maps"
	"sync"
)

const (
	// Fallback number of shards
	DefaultShardsCount = 128
)

type Shard struct {
	mu sync.RWMutex
	m  map[string]string
}

type ShardedMap struct {
	shards []*Shard
	hash   Hasher
}

type Hasher interface {
	Sum64(string) uint64
}

func NewShardedMap(shardsCount int, hasher Hasher) *ShardedMap {
	if shardsCount <= 0 {
		shardsCount = DefaultShardsCount
	}

	shards := make([]*Shard, shardsCount)
	for i := 0; i < shardsCount; i++ {
		shards[i] = &Shard{
			m: make(map[string]string),
		}
	}

	return &ShardedMap{
		shards: shards,
		hash:   hasher,
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
