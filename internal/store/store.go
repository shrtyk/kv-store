package store

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"sync"
	"time"
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
	logger  *slog.Logger
}

func NewStore(l *slog.Logger) *store {
	return &store{
		storage: make(map[string]string),
		logger:  l,
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

func (s *store) StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup, rebuildIn time.Duration) {
	wg.Add(1)
	t := time.NewTicker(rebuildIn)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("map rebuilder stopped")
				return
			case <-t.C:
				newst := make(map[string]string)
				s.mu.Lock()

				maps.Copy(newst, s.storage)
				s.storage = newst

				s.mu.Unlock()
				s.logger.Debug("internal map has been rebuild")
			}
		}
	}()
}
