package store

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewShardedMap(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
	assert.Len(t, m.shards, 16)
	assert.NotNil(t, m.hash)
}

func TestShardedMap_GetShard(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
	shard := m.getShard("test_key")
	assert.NotNil(t, shard)
}

func TestShardedMap_PutAndGet(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})

	m.Put("key1", "value1")
	val, ok := m.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	_, ok = m.Get("non_existent_key")
	assert.False(t, ok)
}

func TestShardedMap_Delete(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
	m.Put("key1", "value1")

	m.Delete("key1")
	_, ok := m.Get("key1")
	assert.False(t, ok)
}

func TestShardedMap_Len(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
	assert.Equal(t, 0, m.Len())

	m.Put("key1", "value1")
	m.Put("key2", "value2")
	assert.Equal(t, 2, m.Len())

	m.Delete("key1")
	assert.Equal(t, 1, m.Len())
}

func TestShardedMap_Items(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
	m.Put("key1", "value1")
	m.Put("key2", "value2")

	items := m.Items()
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	assert.Equal(t, expected, items)
}

func TestShardedMap_ConcurrentAccess(t *testing.T) {
	m := NewShardedMap(16, Xxhasher{})
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
