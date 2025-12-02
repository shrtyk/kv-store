package futures

import (
	"context"
	"errors"
)

var (
	ErrPromiseTimeout = errors.New("promise: timeout exceeded")
)

//go:generate mockery
type FuturesStore interface {
	StartGC(ctx context.Context)
	NewPromise(logIndex int64) Future
	Fulfill(logIndex int64)
}

//go:generate mockery
type Future interface {
	Wait(ctx context.Context) error
}
