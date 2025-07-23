package store

import (
	"context"
	"sync"
)

type Store interface {
	StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup)
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}
