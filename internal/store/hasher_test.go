package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXxhasher_Sum64(t *testing.T) {
	hasher := Xxhasher{}
	hash1 := hasher.Sum64("test_string")
	hash2 := hasher.Sum64("test_string")
	assert.Equal(t, hash1, hash2, "Hashes for the same string should be equal")

	hash3 := hasher.Sum64("another_string")
	assert.NotEqual(t, hash1, hash3, "Hashes for different strings should not be equal")
}
