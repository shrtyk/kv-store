package store

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/cfg"
	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewShardedMap(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	assert.Len(t, m.shards, 16)
	assert.NotNil(t, m.hash)
}

func TestShardedMapGetShard(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	shard := m.getShard("test_key")
	assert.NotNil(t, shard)
}

func TestShardedMapPutAndGet(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})

	m.Put("key1", "value1")
	val, ok := m.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	_, ok = m.Get("non_existent_key")
	assert.False(t, ok)
}

func TestShardedMapDelete(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	m.Put("key1", "value1")

	m.Delete("key1")
	_, ok := m.Get("key1")
	assert.False(t, ok)
}

func TestShardedMapLen(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	assert.Equal(t, 0, m.Len())

	m.Put("key1", "value1")
	m.Put("key2", "value2")
	assert.Equal(t, 2, m.Len())

	m.Delete("key1")
	assert.Equal(t, 1, m.Len())
}

func TestShardedMapItems(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	m.Put("key1", "value1")
	m.Put("key2", "value2")

	items := m.Items()
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	assert.Equal(t, expected, items)
}

func TestShardedMapConcurrentAccess(t *testing.T) {
	m := NewShardedMap(tu.NewMockShardsCfg(), 16, Xxhasher{})
	var wg sync.WaitGroup

	// Concurrent Puts
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			value := "value" + strconv.Itoa(i)
			m.Put(key, value)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 100, m.Len())

	// Concurrent Gets
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			val, ok := m.Get(key)
			assert.True(t, ok)
			assert.Equal(t, "value"+strconv.Itoa(i), val)
		}(i)
	}
	wg.Wait()

	// Concurrent Deletes
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "key" + strconv.Itoa(i)
			m.Delete(key)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 50, m.Len())
}

func TestShardRebuild(t *testing.T) {
	shardsCfg := &cfg.ShardsCfg{
		SparseRatio:        0.5,
		MinOpsUntilRebuild: 200,
		MinDeletes:         100,
	}
	shard := &Shard{
		cfg: shardsCfg,
		m:   make(map[string]string),
	}

	for i := range 200 {
		shard.m["key"+strconv.Itoa(i)] = "value"
	}
	shard.puts = 200
	shard.maxSize = 200

	for i := range 100 {
		delete(shard.m, "key"+strconv.Itoa(i))
	}
	shard.deletes = 100

	assert.True(t, shard.needsRebuild())

	shard.rebuild()

	assert.Equal(t, uint64(0), shard.puts)
	assert.Equal(t, uint64(0), shard.deletes)
	assert.Equal(t, 100, shard.maxSize)
	assert.Len(t, shard.m, 100)
}

type mockHasher struct{}

func (h *mockHasher) Sum64(s string) uint64 {
	return 0
}

func TestShardsSupervisor(t *testing.T) {
	shardsCfg := &cfg.ShardsCfg{
		CheckFreq:          10 * time.Millisecond,
		SparseRatio:        0.5,
		MinOpsUntilRebuild: 200,
		MinDeletes:         100,
		WorkersCount:       4,
	}
	m := NewShardedMap(shardsCfg, 1, &mockHasher{})
	shard := m.shards[0]

	for i := range 200 {
		m.Put("key"+strconv.Itoa(i), "value")
	}

	for i := range 100 {
		m.Delete("key" + strconv.Itoa(i))
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	m.StartShardsSupervisor(ctx, &wg)

	assert.Eventually(t, func() bool {
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return shard.puts == 0 && shard.deletes == 0
	}, 100*time.Millisecond, 10*time.Millisecond, "shard was not rebuilt")

	cancel()
	wg.Wait()

	shard.mu.RLock()
	assert.Equal(t, uint64(0), shard.puts)
	assert.Equal(t, uint64(0), shard.deletes)
	assert.Equal(t, 100, shard.maxSize)
	assert.Len(t, shard.m, 100)
	shard.mu.RUnlock()
}
