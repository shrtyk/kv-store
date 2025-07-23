package store

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/shrtyk/kv-store/pkg/cfg"
)

var (
	ErrNoSuchKey     = errors.New("no such key")
	ErrKeyTooLarge   = errors.New("key too large")
	ErrValueTooLarge = errors.New("value too large")
)

type store struct {
	cfg       *cfg.StoreCfg
	mu        sync.RWMutex
	storage   map[string]string
	logger    *slog.Logger
	puts      int
	deletions int
	maxSize   int
}

func NewStore(cfg *cfg.StoreCfg, l *slog.Logger) *store {
	return &store{
		cfg:     cfg,
		storage: make(map[string]string),
		logger:  l,
	}
}

func (s *store) Put(key, value string) error {
	if len(key) > s.cfg.MaxKeySize {
		return ErrKeyTooLarge
	}
	if len(value) > s.cfg.MaxValSize {
		return ErrValueTooLarge
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.storage[key]
	s.storage[key] = value
	if !ok {
		s.puts++
	}
	s.maxSize = max(s.maxSize, len(s.storage))
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

	_, ok := s.storage[key]
	delete(s.storage, key)
	if ok {
		s.deletions++
	}
	return nil
}

func (s *store) StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup) {
	t := time.NewTicker(s.cfg.TryRebuildIn)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				s.logger.Info("map rebuilder stopped")
				return
			case <-t.C:
				s.mu.RLock()
				deletions := s.deletions
				maxSize := s.maxSize
				puts := s.puts
				curSize := len(s.storage)
				s.mu.RUnlock()

				totalOps := puts + deletions
				// Map will be rebuilt only if all conditions are true:
				// 1. the number of deletions since the last rebuild has reached a minimum threshold
				// 2. the current map size is significantly smaller than its peak size
				// 3. a minimum number of total operations have occurred
				if deletions >= s.cfg.MinDeletesTrigger &&
					curSize <= int(float64(maxSize)*s.cfg.SparseRatio) &&
					totalOps >= s.cfg.MinOpsBeforeRebuild {
					s.rebuildInternalMap(curSize)
					s.logger.Debug("internal map has been rebuilt")
				}
			}
		}
	}()
}

func (s *store) rebuildInternalMap(newSize int) {
	newst := make(map[string]string, newSize)
	s.mu.Lock()
	defer s.mu.Unlock()

	maps.Copy(newst, s.storage)
	s.storage = newst
	s.puts = 0
	s.deletions = 0
	s.maxSize = newSize
}
