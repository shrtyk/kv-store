package store

import (
	"errors"
	"sync"

	"github.com/shrtyk/kv-store/internal/tlog"
)

var (
	ErrorNoSuchKey = errors.New("no such key")
)

type store struct {
	mu      sync.RWMutex
	storage map[string]string
	tl      tlog.TransactionsLogger
}

func NewStore(tl tlog.TransactionsLogger) *store {
	return &store{
		storage: make(map[string]string),
		tl:      tl,
	}
}

func (s *store) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tl.WritePut(key, value)
	s.storage[key] = value
	return nil
}

func (s *store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.storage[key]
	if !ok {
		return "", ErrorNoSuchKey
	}
	return val, nil
}

func (s *store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tl.WriteDelete(key)
	delete(s.storage, key)
	return nil
}
