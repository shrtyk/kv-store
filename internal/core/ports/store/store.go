package store

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrNoSuchKey     = errors.New("no such key")
	ErrKeyTooLarge   = errors.New("key too large")
	ErrValueTooLarge = errors.New("value too large")
)

//go:generate mockery
type Store interface {
	StartMapRebuilder(ctx context.Context, wg *sync.WaitGroup)
	Put(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	Items() map[string]string
}