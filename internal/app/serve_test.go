package app

import (
	"io"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	tu "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	l, _ := tu.NewMockLogger()
	lcfg := tu.NewMockTransLogCfg()
	tu.FileCleanUp(t, lcfg.LogFileName)

	app := NewApp()
	tl := tlog.MustCreateNewFileTransLog(lcfg, l)
	defer tl.Close()

	store := store.NewStore(tu.NewMockStoreCfg(), l)
	app.Init(
		WithStore(store),
		WithTransactionalLogger(tl),
		WithLogger(l),
	)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		app.Serve("localhost:8081")
	}()

	var (
		err     error
		retries = 10
		r       *http.Response
	)
	for range retries {
		r, err = http.Get("http://localhost:8081/v1")
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("server did not start in time: %v", err)
	}
	defer r.Body.Close()

	assert.Equal(t, http.StatusOK, r.StatusCode)
	b, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	assert.Equal(t, "Hello!\n", string(b))

	p, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	p.Signal(syscall.SIGTERM)

	wg.Wait()
}
