//go:build performance
// +build performance

package app

import (
	"bytes"
	"net/http"
	"sync"
	"testing"
	"time"

	tutils "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/stretchr/testify/require"
)

func TestFunctional_BulkPutDeletePerformance(t *testing.T) {
	const (
		baseURL        = "http://localhost:16700/v1"
		putRequests    = 1000000
		deleteRequests = 666666
		keySize        = 100
		valueSize      = 100
		workersCount   = 1000
	)

	transport := &http.Transport{
		MaxIdleConns:        workersCount,
		MaxIdleConnsPerHost: workersCount,
	}
	client := &http.Client{Transport: transport}

	keys := make([]string, putRequests)
	values := make([]string, putRequests)
	for i := 0; i < putRequests; i++ {
		keys[i] = tutils.RandomString(t, keySize)
		values[i] = tutils.RandomString(t, valueSize)
	}

	t.Logf("Starting performance test with %d workers...", workersCount)

	// --- PUT Phase ---
	t.Logf("Starting concurrent bulk PUT test with %d requests...", putRequests)

	putJobs := make(chan int, putRequests)
	var wgPut sync.WaitGroup

	for w := 0; w < workersCount; w++ {
		wgPut.Add(1)
		go func() {
			defer wgPut.Done()
			for i := range putJobs {
				key := keys[i]
				value := values[i]
				url := baseURL + "/" + key

				req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(value))
				require.NoError(t, err)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Equal(t, http.StatusCreated, resp.StatusCode)
			}
		}()
	}

	putStart := time.Now()

	for i := 0; i < putRequests; i++ {
		putJobs <- i
	}
	close(putJobs)

	wgPut.Wait()
	putDuration := time.Since(putStart)
	t.Logf("Finished %d PUT requests in %s", putRequests, putDuration)
	t.Logf("Average throughput (PUT): %.2f req/s", float64(putRequests)/putDuration.Seconds())

	// --- DELETE Phase ---
	t.Logf("Starting concurrent bulk DELETE test with %d requests...", workersCount)

	deleteJobs := make(chan int, deleteRequests)
	var wgDelete sync.WaitGroup

	for w := 0; w < workersCount; w++ {
		wgDelete.Add(1)
		go func() {
			defer wgDelete.Done()
			for i := range deleteJobs {
				key := keys[i]
				url := baseURL + "/" + key

				req, err := http.NewRequest(http.MethodDelete, url, nil)
				require.NoError(t, err)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}()
	}

	deleteStart := time.Now()

	for i := 0; i < deleteRequests; i++ {
		deleteJobs <- i
	}
	close(deleteJobs)

	wgDelete.Wait()
	deleteDuration := time.Since(deleteStart)
	t.Logf("Finished %d DELETE requests in %s", deleteRequests, deleteDuration)
	t.Logf("Average throughput (DELETE): %.2f req/s", float64(deleteRequests)/deleteDuration.Seconds())
}
