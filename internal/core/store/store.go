package store

import (
	"context"
	"log/slog"
	"sync"

	"github.com/shrtyk/kv-store/internal/cfg"
	pstore "github.com/shrtyk/kv-store/internal/core/ports/store"
)

type store struct {
	cfg     *cfg.StoreCfg
	storage *ShardedMap

	logger *slog.Logger
}

func NewStore(wg *sync.WaitGroup, cfg *cfg.StoreCfg, shardCfg *cfg.ShardsCfg, l *slog.Logger) *store {
	return &store{
		cfg:     cfg,
		storage: NewShardedMap(shardCfg, shardCfg.ShardsCount, Xxhasher{}),
		logger:  l,
	}
}

func (s *store) Put(key, value string) error {
	if len(key) > s.cfg.MaxKeySize {
		return pstore.ErrKeyTooLarge
	}
	if len(value) > s.cfg.MaxValSize {
		return pstore.ErrValueTooLarge
	}
	s.storage.Put(key, value)
	return nil
}

func (s *store) Get(key string) (string, error) {
	val, ok := s.storage.Get(key)
	if !ok {
		return "", pstore.ErrNoSuchKey
	}
	return val, nil
}

func (s *store) Delete(key string) error {
	s.storage.Delete(key)
	return nil
}

func (s *store) StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup) {
	s.storage.StartShardsSupervisor(ctx, wg)
}

func (s *store) Items() map[string]string {
	return s.storage.Items()
}

func (s *store) RestoreFromSnapshot(snapData map[string]string) {
	s.storage.RestoreFromSnapshot(snapData)
}
