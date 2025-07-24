package tlog

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/store"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockstore struct{}

func (m *mockstore) Put(key, value string) error {
	return nil
}
func (m *mockstore) Get(key string) (string, error) {
	return "", nil
}
func (m *mockstore) Delete(key string) error {
	return nil
}

func TestTransactionFileLoggger(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	k, v := "test-key", "test-val"
	tl, err := NewFileTransactionalLogger(lcfg, l)
	require.NoError(t, err)
	defer tl.Close()

	tl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	tl.WritePut(k, v)
	tl.WritePut(k, v)

	events, errs := tl.ReadEvents()
	for e := range events {
		assert.EqualValues(t, k, e.key)
		assert.EqualValues(t, v, e.value)
	}
	assert.NoError(t, <-errs)
	tl.WaitWritings()

	ntl := MustCreateNewFileTransLog(lcfg, l)
	defer ntl.Close()
	ntl.Start(context.Background(), &sync.WaitGroup{}, &mockstore{})

	events, errs = ntl.ReadEvents()
	for range events {
	}
	assert.NoError(t, <-errs)

	ntl.WritePut(k, v)
	ntl.WritePut(k, v)
	ntl.WaitWritings()
	assert.EqualValues(t, uint64(4), ntl.lastSeq)
}

func TestTransactionLoggerCompacting(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	s := store.NewStore(tu.NewMockStoreCfg(), l)

	tl, err := NewFileTransactionalLogger(lcfg, l)
	require.NoError(t, err)
	defer tl.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	tl.Start(ctx, &sync.WaitGroup{}, s)

	for i := range 10 {
		tl.WritePut(strconv.Itoa(i), strconv.Itoa(i+1))
	}

	for i := range 5 {
		tl.WriteDelete(strconv.Itoa(i))
	}

	tl.WaitWritings()

	tl.Compact()

	assert.Eventually(
		t,
		func() bool { return tl.lastSeq == 5 },
		500*time.Millisecond,
		20*time.Millisecond,
	)
}
