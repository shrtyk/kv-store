package httphandlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/store"
	"github.com/shrtyk/kv-store/internal/tlog"
	tutils "github.com/shrtyk/kv-store/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	testName string
	method   string
	key      string
	value    string
	wantBody string
	status   int
}

type subtest func(t *testing.T)

func subTestTemplate(router http.Handler, c testCase) subtest {
	return func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := makeReqBasedOnMethod(t, c.method, c.key, c.value)
		router.ServeHTTP(rr, req)

		b, err := io.ReadAll(rr.Body)
		assert.NoError(t, err)
		assert.EqualValues(t, c.wantBody, string(b))
		statusCodeAssertion(t, rr.Result().StatusCode, c.status)

		assert.NoError(t, req.Body.Close())
	}
}

func NewTestRouter(s store.Store, tl tlog.TransactionsLogger) *chi.Mux {
	hh := NewHandlersProvider(s, tl)
	mux := chi.NewMux()
	mux.Route("/v1", func(r chi.Router) {
		r.Get("/", hh.HelloHandler)
		r.Put("/{key}", hh.PutHandler)
		r.Get("/{key}", hh.GetHandler)
		r.Delete("/{key}", hh.DeleteHandler)
	})
	return mux
}

func TestHandlers(t *testing.T) {
	testFileName := tutils.FileNameWithCleanUp(t, "test")

	k, v := "test-key", "test-val"
	tl := tlog.MustCreateNewFileTransLog(testFileName)

	store := store.NewStore()
	tl.Start(t.Context(), store)
	router := NewTestRouter(store, tl)

	testCases := []testCase{
		{
			testName: "get key in empty store",
			key:      k,
			method:   http.MethodGet,
			wantBody: "404 page not found\n",
			status:   http.StatusNotFound,
		},
		{
			testName: "put initial key",
			key:      k,
			value:    v,
			method:   http.MethodPut,
			status:   http.StatusCreated,
		},
		{
			testName: "delete wrong key",
			key:      "test-key1",
			method:   http.MethodDelete,
			status:   http.StatusOK,
		},
		{
			testName: "delete right key",
			key:      k,
			method:   http.MethodDelete,
			status:   http.StatusOK,
		},
		{
			testName: "try to get deleted key",
			key:      k,
			method:   http.MethodGet,
			wantBody: "404 page not found\n",
			status:   http.StatusNotFound,
		},
	}

	for _, c := range testCases {
		t.Run(c.testName, subTestTemplate(router, c))
	}
	assert.NoError(t, tl.Close())
}

type mockStore struct {
	errOnPut    error
	errOnGet    error
	errOnDelete error
}

func (m *mockStore) Put(key, value string) error {
	return m.errOnPut
}

func (m *mockStore) Get(key string) (string, error) {
	return "", m.errOnGet
}

func (m *mockStore) Delete(key string) error {
	return m.errOnDelete
}

func TestInternalErrWithMocks(t *testing.T) {
	msg := "a simulated store error"
	mockErr := errors.New(msg)
	s := &mockStore{
		errOnPut:    mockErr,
		errOnGet:    mockErr,
		errOnDelete: mockErr,
	}

	fileName := tutils.FileNameWithCleanUp(t, "test")
	k, v := "any-key", "any-val"
	tl := tlog.MustCreateNewFileTransLog(fileName)
	tl.Start(t.Context(), s)
	mockRouter := NewTestRouter(s, tl)

	errorTestCases := []struct {
		testName string
		method   string
		key      string
		value    string
		wantBody string
		status   int
	}{
		{
			testName: "put returns internal server error",
			method:   http.MethodPut,
			key:      k,
			value:    v,
			status:   http.StatusInternalServerError,
			wantBody: msg + "\n",
		},
		{
			testName: "get returns internal server error",
			method:   http.MethodGet,
			key:      k,
			status:   http.StatusInternalServerError,
			wantBody: msg + "\n",
		},
		{
			testName: "delete returns internal server error",
			method:   http.MethodDelete,
			key:      k,
			status:   http.StatusInternalServerError,
			wantBody: msg + "\n",
		},
	}

	for _, c := range errorTestCases {
		t.Run(c.testName, subTestTemplate(mockRouter, c))
	}
	assert.NoError(t, tl.Close())
}

func makeReqBasedOnMethod(t testing.TB, method string, key, val string) *http.Request {
	t.Helper()

	var r *http.Request
	switch method {
	case http.MethodGet:
		if key == "" {
			r = httptest.NewRequest(method, "/v1", nil)
		} else {
			r = httptest.NewRequest(method, "/v1/"+key, nil)
		}
	case http.MethodPut:
		body := bytes.NewReader([]byte(val))
		r = httptest.NewRequest(method, "/v1/"+key, body)
	case http.MethodDelete:
		r = httptest.NewRequest(method, "/v1/"+key, nil)
	default:
		t.Error("unknown method")
	}
	return r
}

func statusCodeAssertion(t testing.TB, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
