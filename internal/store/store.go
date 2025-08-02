package store

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/shrtyk/kv-store/pkg/cfg"
)

var (
	ErrNoSuchKey     = errors.New("no such key")
	ErrKeyTooLarge   = errors.New("key too large")
	ErrValueTooLarge = errors.New("value too large")
)

type store struct {
	cfg     *cfg.StoreCfg
	storage *ShardedMap

	logger *slog.Logger
}

func NewStore(cfg *cfg.StoreCfg, l *slog.Logger) *store {
	return &store{
		cfg:     cfg,
		storage: NewShardedMap(cfg.ShardsCount, Xxhasher{}),
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
	s.storage.Put(key, value)
	return nil
}

func (s *store) Get(key string) (string, error) {
	val, ok := s.storage.Get(key)
	if !ok {
		return "", ErrNoSuchKey
	}
	return val, nil
}

func (s *store) Delete(key string) error {
	s.storage.Delete(key)
	return nil
}

func (s *store) StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup) {
	// TODO
}

func (s *store) Items() map[string]string {
	return s.storage.Items()
}
