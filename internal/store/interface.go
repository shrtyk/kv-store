package store

import (
	"context"
	"sync"
	"time"
)

type Store interface {
	StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup, rebuildIn time.Duration)
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}
