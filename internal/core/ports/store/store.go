package store

import (
	"context"
	"sync"
)

//go:generate mockery
type Store interface {
	StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup)
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Items() map[string]string
}
