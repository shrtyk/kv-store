package tlog

import (
	"context"
	"sync"

	"github.com/shrtyk/kv-store/internal/core/ports/store"
	pb "github.com/shrtyk/kv-store/proto/log_entries/gen"
)

//go:generate mockery
type TransactionsLogger interface {
	Start(ctx context.Context, wg *sync.WaitGroup, s store.Store)
	WritePut(key, val string)
	WriteDelete(key string)
	WaitWritings()

	ReadEvents() (<-chan *pb.LogEntry, <-chan error)
	Err() <-chan error
	Close() error
}
