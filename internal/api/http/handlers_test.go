package httphandlers

// import (
// 	"bytes"
// 	"errors"
// 	"io"
// 	"net/http"
// 	"net/http/httptest"
// 	"sync"
// 	"testing"

// 	"github.com/go-chi/chi/v5"
// 	pstore "github.com/shrtyk/kv-store/internal/core/ports/store"
// 	storemocks "github.com/shrtyk/kv-store/internal/core/ports/store/mocks"
// 	ptlog "github.com/shrtyk/kv-store/internal/core/ports/tlog"
// 	tlogmocks "github.com/shrtyk/kv-store/internal/core/ports/tlog/mocks"
// 	"github.com/shrtyk/kv-store/internal/core/snapshot"
// 	"github.com/shrtyk/kv-store/internal/core/store"
// 	"github.com/shrtyk/kv-store/internal/core/tlog"
// 	pmts "github.com/shrtyk/kv-store/internal/infrastructure/prometheus"
// 	tu "github.com/shrtyk/kv-store/internal/tests/testutils"
// 	"github.com/stretchr/testify/assert"
// )

// type testCase struct {
// 	testName string
// 	method   string
// 	key      string
// 	value    string
// 	wantBody string
// 	status   int
// }

// type subtest func(t *testing.T)

// func subTestTemplate(router http.Handler, c testCase) subtest {
// 	return func(t *testing.T) {
// 		rr := httptest.NewRecorder()
// 		req := makeReqBasedOnMethod(t, c.method, c.key, c.value)
// 		router.ServeHTTP(rr, req)

// 		b, err := io.ReadAll(rr.Body)
// 		assert.NoError(t, err)
// 		assert.EqualValues(t, c.wantBody, string(b))
// 		statusCodeAssertion(t, rr.Result().StatusCode, c.status)

// 		assert.NoError(t, req.Body.Close())
// 	}
// }

// func NewTestRouter(s pstore.Store, tl ptlog.TransactionsLogger) *chi.Mux {
// 	hh := NewHandlersProvider(
// 		tu.NewMockStoreCfg(),
// 		s,
// 		tl,
// 		pmts.NewMockMetrics(),
// 	)
// 	mux := chi.NewMux()
// 	mux.Get("/healthz", hh.Healthz)
// 	mux.Route("/v1", func(r chi.Router) {
// 		r.Put("/{key}", hh.PutHandler)
// 		r.Get("/{key}", hh.GetHandler)
// 		r.Delete("/{key}", hh.DeleteHandler)
// 	})
// 	return mux
// }

// func TestHandlers(t *testing.T) {
// 	l, _ := tu.NewMockLogger()
// 	lcfg := tu.NewMockTransLogCfg()
// 	tu.FileCleanUp(t, lcfg.LogFileName)

// 	k, v := "test-key", "test-val"
// 	snapshotter := snapshot.NewFileSnapshotter(
// 		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
// 		l,
// 	)
// 	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)

// 	appstore := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)
// 	tl.Start(t.Context(), &sync.WaitGroup{}, appstore)
// 	router := NewTestRouter(appstore, tl)

// 	testCases := []testCase{
// 		{
// 			testName: "get key in empty store",
// 			key:      k,
// 			method:   http.MethodGet,
// 			wantBody: "404 page not found\n",
// 			status:   http.StatusNotFound,
// 		},
// 		{
// 			testName: "put initial key",
// 			key:      k,
// 			value:    v,
// 			method:   http.MethodPut,
// 			status:   http.StatusCreated,
// 		},
// 		{
// 			testName: "delete wrong key",
// 			key:      "test-key1",
// 			method:   http.MethodDelete,
// 			status:   http.StatusOK,
// 		},
// 		{
// 			testName: "delete right key",
// 			key:      k,
// 			method:   http.MethodDelete,
// 			status:   http.StatusOK,
// 		},
// 		{
// 			testName: "try to get deleted key",
// 			key:      k,
// 			method:   http.MethodGet,
// 			wantBody: "404 page not found\n",
// 			status:   http.StatusNotFound,
// 		},
// 		{
// 			testName: "try to put huge key",
// 			key:      tu.RandomString(t, 101),
// 			method:   http.MethodPut,
// 			wantBody: "key too large\n",
// 			status:   http.StatusBadRequest,
// 		},
// 		{
// 			testName: "try to put huge val",
// 			key:      k,
// 			value:    tu.RandomString(t, 101),
// 			method:   http.MethodPut,
// 			wantBody: "value too large\n",
// 			status:   http.StatusBadRequest,
// 		},
// 	}

// 	for _, c := range testCases {
// 		t.Run(c.testName, subTestTemplate(router, c))
// 	}
// 	assert.NoError(t, tl.Close())
// }

// func TestInternalErrWithMocks(t *testing.T) {
// 	msg := "a simulated store error"
// 	mockErr := errors.New(msg)

// 	k, v := "any-key", "any-val"

// 	mockStore := storemocks.NewMockStore(t)
// 	mockStore.EXPECT().Put(k, v).Return(mockErr).Maybe()
// 	mockStore.EXPECT().Get(k).Return("", mockErr).Maybe()
// 	mockStore.EXPECT().Delete(k).Return(mockErr).Maybe()

// 	mockTLog := tlogmocks.NewMockTransactionsLogger(t)
// 	mockTLog.EXPECT().WritePut(k, v).Return().Maybe()
// 	mockTLog.EXPECT().WriteDelete(k).Return().Maybe()

// 	mockRouter := NewTestRouter(mockStore, mockTLog)

// 	errorTestCases := []struct {
// 		testName string
// 		method   string
// 		key      string
// 		value    string
// 		wantBody string
// 		status   int
// 	}{
// 		{
// 			testName: "put returns internal server error",
// 			method:   http.MethodPut,
// 			key:      k,
// 			value:    v,
// 			status:   http.StatusInternalServerError,
// 			wantBody: msg + "\n",
// 		},
// 		{
// 			testName: "get returns internal server error",
// 			method:   http.MethodGet,
// 			key:      k,
// 			status:   http.StatusInternalServerError,
// 			wantBody: msg + "\n",
// 		},
// 		{
// 			testName: "delete returns internal server error",
// 			method:   http.MethodDelete,
// 			key:      k,
// 			status:   http.StatusInternalServerError,
// 			wantBody: msg + "\n",
// 		},
// 	}

// 	for _, c := range errorTestCases {
// 		t.Run(c.testName, subTestTemplate(mockRouter, c))
// 	}
// }

// func TestHealthz(t *testing.T) {
// 	router := testRouter(t)

// 	rr := httptest.NewRecorder()
// 	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)

// 	router.ServeHTTP(rr, healthReq)

// 	b, err := io.ReadAll(rr.Body)
// 	assert.NoError(t, err)
// 	assert.EqualValues(t, "kv-store up and healthy", string(b))
// 	statusCodeAssertion(t, rr.Result().StatusCode, http.StatusOK)

// 	assert.NoError(t, healthReq.Body.Close())
// }

// func testRouter(t *testing.T) *chi.Mux {
// 	t.Helper()

// 	l, _ := tu.NewMockLogger()
// 	lcfg := tu.NewMockTransLogCfg()
// 	tu.FileCleanUp(t, lcfg.LogFileName)

// 	snapshotter := snapshot.NewFileSnapshotter(
// 		tu.NewMockSnapshotsCfg(t.TempDir(), 2),
// 		l,
// 	)
// 	tl := tlog.MustCreateNewFileTransLog(lcfg, l, snapshotter)

// 	appstore := store.NewStore(&sync.WaitGroup{}, tu.NewMockStoreCfg(), tu.NewMockShardsCfg(), l)
// 	tl.Start(t.Context(), &sync.WaitGroup{}, appstore)
// 	return NewTestRouter(appstore, tl)
// }

// func makeReqBasedOnMethod(t testing.TB, method string, key, val string) *http.Request {
// 	t.Helper()

// 	var r *http.Request
// 	switch method {
// 	case http.MethodGet:
// 		if key == "" {
// 			r = httptest.NewRequest(method, "/v1", nil)
// 		} else {
// 			r = httptest.NewRequest(method, "/v1/"+key, nil)
// 		}
// 	case http.MethodPut:
// 		body := bytes.NewReader([]byte(val))
// 		r = httptest.NewRequest(method, "/v1/"+key, body)
// 	case http.MethodDelete:
// 		r = httptest.NewRequest(method, "/v1/"+key, nil)
// 	default:
// 		t.Error("unknown method")
// 	}
// 	return r
// }

// func statusCodeAssertion(t testing.TB, got, want int) {
// 	t.Helper()
// 	if got != want {
// 		t.Errorf("got: %v, want: %v", got, want)
// 	}
// }
