package httphandlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shrtyk/kv-store/internal/cfg"
	futuresmocks "github.com/shrtyk/kv-store/internal/core/ports/futures/mocks"
	metricsmocks "github.com/shrtyk/kv-store/internal/core/ports/metrics/mocks"
	"github.com/shrtyk/kv-store/internal/core/ports/store"
	storemocks "github.com/shrtyk/kv-store/internal/core/ports/store/mocks"
	rmocks "github.com/shrtyk/kv-store/internal/core/raft/mocks"
	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type handlerSetup struct {
	hp          *handlersProvider
	mockStore   *storemocks.MockStore
	stubRaft    *rmocks.StubRaft
	mockFutures *futuresmocks.MockFuturesStore
	mockMetrics *metricsmocks.MockMetrics
	mockFuture  *futuresmocks.MockFuture
}

func setup(t *testing.T) handlerSetup {
	stCfg := &cfg.StoreCfg{MaxKeySize: 10, MaxValSize: 20}
	mockStore := storemocks.NewMockStore(t)
	stubRaft := rmocks.NewStubRaft(mockStore, true, 1)
	mockFutures := futuresmocks.NewMockFuturesStore(t)
	mockMetrics := metricsmocks.NewMockMetrics(t)
	mockFuture := futuresmocks.NewMockFuture(t)
	addrs := []string{"http://follower:8080", "http://leader:8080"}

	hp := NewHandlersProvider(
		stCfg,
		mockStore,
		mockMetrics,
		stubRaft,
		mockFutures,
		addrs,
	)

	return handlerSetup{hp, mockStore, stubRaft, mockFutures, mockMetrics, mockFuture}
}

func TestPutHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key, value := "testkey", "testvalue"

		s.mockStore.On("Put", key, value).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(nil).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()
		s.mockMetrics.On("HttpPut", key, mock.Anything).Return().Once()

		req := httptest.NewRequest(http.MethodPut, "/v1/"+key, strings.NewReader(value))
		rr := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.PutHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		s.mockStore.AssertExpectations(t)
		s.mockMetrics.AssertExpectations(t)
		s.mockFutures.AssertExpectations(t)
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)
		key, value := "testkey", "testvalue"

		req := httptest.NewRequest(http.MethodPut, "/v1/"+key, strings.NewReader(value))
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.PutHandler(rr, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
		assert.Equal(t, "http://leader:8080/v1/"+key, rr.Header().Get("Location"))
	})

	t.Run("key too large", func(t *testing.T) {
		s := setup(t)
		key, value := "thiskeyistoolarge", "value"

		req := httptest.NewRequest(http.MethodPut, "/v1/"+key, strings.NewReader(value))
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.PutHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("value too large", func(t *testing.T) {
		s := setup(t)
		key, value := "key", "thisvalueistoolargetoomuch"

		req := httptest.NewRequest(http.MethodPut, "/v1/"+key, strings.NewReader(value))
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.PutHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("future timeout", func(t *testing.T) {
		s := setup(t)
		key, value := "testkey", "testvalue"

		s.mockStore.On("Put", key, value).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(context.DeadlineExceeded).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()

		req := httptest.NewRequest(http.MethodPut, "/v1/"+key, strings.NewReader(value))
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.PutHandler(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	})
}

func TestGetHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key, value := "testkey", "testvalue"

		s.stubRaft.SetReadOnlyResult([]byte(value), nil)
		s.mockMetrics.On("HttpGet", key, mock.Anything).Return().Once()

		req := httptest.NewRequest(http.MethodGet, "/v1/"+key, nil)
		rr := httptest.NewRecorder()

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.GetHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, value, rr.Body.String())
		s.mockMetrics.AssertExpectations(t)
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)
		key := "testkey"

		req := httptest.NewRequest(http.MethodGet, "/v1/"+key, nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.GetHandler(rr, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
		assert.Equal(t, "http://leader:8080/v1/"+key, rr.Header().Get("Location"))
	})

	t.Run("not found", func(t *testing.T) {
		s := setup(t)
		key := "notfound"

		s.stubRaft.SetReadOnlyResult(nil, store.ErrNoSuchKey)

		req := httptest.NewRequest(http.MethodGet, "/v1/"+key, nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.GetHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestDeleteHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := setup(t)
		key := "testkey"

		s.mockStore.On("Delete", key).Return(nil).Once()
		s.mockFuture.On("Wait", mock.Anything).Return(nil).Once()
		s.mockFutures.On("NewFuture", mock.Anything).Return(s.mockFuture).Once()
		s.mockMetrics.On("HttpDelete", key, mock.Anything).Return().Once()

		req := httptest.NewRequest(http.MethodDelete, "/v1/"+key, nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.DeleteHandler(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		s.mockStore.AssertExpectations(t)
		s.mockMetrics.AssertExpectations(t)
	})

	t.Run("not leader", func(t *testing.T) {
		s := setup(t)
		s.stubRaft.SetLeader(false)
		key := "testkey"

		req := httptest.NewRequest(http.MethodDelete, "/v1/"+key, nil)
		rr := httptest.NewRecorder()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("key", key)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		req = req.WithContext(logger.ToCtx(req.Context(), logger.NewLogger("dev")))

		s.hp.DeleteHandler(rr, req)

		assert.Equal(t, http.StatusTemporaryRedirect, rr.Code)
	})
}
