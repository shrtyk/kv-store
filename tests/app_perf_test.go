package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func randomString(t *testing.T, size int) string {
	t.Helper()
	b := make([]byte, size)
	_, err := rand.Read(b)
	require.NoError(t, err, "Failed to generate random bytes")
	return hex.EncodeToString(b)
}

func TestFunctional_BulkPutDeletePerformance(t *testing.T) {
	const (
		baseURL        = "http://localhost:16700/v1"
		putRequests    = 10000
		deleteRequests = 6666
		keySize        = 50
		valueSize      = 50
	)

	client := &http.Client{}
	keysCreated := make([]string, 0, putRequests)

	// PUT Phase
	t.Logf("Starting bulk PUT test with %d requests...", putRequests)
	putStart := time.Now()

	for range putRequests {
		key := randomString(t, keySize)
		value := randomString(t, valueSize)
		url := baseURL + "/" + key

		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(value))
		require.NoError(t, err, "Failed to create request")

		resp, err := client.Do(req)
		require.NoError(t, err, "Request failed")
		resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode, "Expected status 201 Created for key %s", key)
		keysCreated = append(keysCreated, key)
	}

	putDuration := time.Since(putStart)
	t.Logf("Finished %d PUT requests in %s", putRequests, putDuration)
	t.Logf("Average time per PUT request: %s", putDuration/time.Duration(putRequests))

	// DELETE Phase
	t.Logf("Starting bulk DELETE test with %d requests...", deleteRequests)
	deleteStart := time.Now()

	for i := range deleteRequests {
		key := keysCreated[i]
		url := baseURL + "/" + key

		req, err := http.NewRequest(http.MethodDelete, url, nil)
		require.NoError(t, err, "Failed to create request")

		resp, err := client.Do(req)
		require.NoError(t, err, "Request failed")
		resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200 OK for deleting key %s", key)
	}

	deleteDuration := time.Since(deleteStart)
	t.Logf("Finished %d DELETE requests in %s", deleteRequests, deleteDuration)
	t.Logf("Average time per DELETE request: %s", deleteDuration/time.Duration(deleteRequests))
}
