package store

import (
	"github.com/cespare/xxhash/v2"
)

// Hasher that uses the xxhash algorithm.
type Xxhasher struct{}

// Sum64 returns the hash sum of the given string.
func (h Xxhasher) Sum64(s string) uint64 {
	return xxhash.Sum64String(s)
}
