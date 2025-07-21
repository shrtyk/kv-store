package httphandlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shrtyk/kv-store/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestHandlers(t *testing.T) {
	store := store.NewStore()
	hh := NewHandlersProvider(store)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	hh.HelloHandler(rr, req)
	assert.EqualValues(t, http.StatusOK, rr.Result().StatusCode)

	b, err := io.ReadAll(rr.Body)
	assert.NoError(t, err)
	assert.EqualValues(t, "Hello!\n", string(b))
}
