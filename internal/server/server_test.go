package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHttpServer(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	helloHandler(rr, req)
	assert.EqualValues(t, http.StatusOK, rr.Result().StatusCode)

	b, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.EqualValues(t, "Hello!\n", string(b))
}
