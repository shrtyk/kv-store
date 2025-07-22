package store

import (
	"errors"
	"sync"
)

var (
	ErrNoSuchKey     = errors.New("no such key")
	ErrKeyTooLarge   = errors.New("key too large")
	ErrValueTooLarge = errors.New("value too large")
)

const (
	maxKeySize = 512
	maxValSize = 512
)

type store struct {
	mu      sync.RWMutex
	storage map[string]string
}

func NewStore() *store {
	return &store{
		storage: make(map[string]string),
	}
}

func (s *store) Put(key, value string) error {
	if len(key) > maxKeySize {
		return ErrKeyTooLarge
	}
	if len(value) > maxValSize {
		return ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.storage[key] = value
	return nil
}

func (s *store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.storage[key]
	if !ok {
		return "", ErrNoSuchKey
	}
	return val, nil
}

func (s *store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.storage, key)
	return nil
}
