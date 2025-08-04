package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type mockMetrics struct {
	called  bool
	code    int
	method  string
	path    string
	latency float64
}

func (m *mockMetrics) HttpRequest(code int, method, path string, latency float64) {
	m.called = true
	m.code = code
	m.method = method
	m.path = path
	m.latency = latency
}
func (m *mockMetrics) Put(key string, duration float64)    {}
func (m *mockMetrics) Delete(key string, duration float64) {}
func (m *mockMetrics) Get(key string, duration float64)    {}

func TestHttpMetrics(t *testing.T) {
	mockMetrics := &mockMetrics{}
	mws := NewMiddlewares(mockMetrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("test"))
	})

	router := chi.NewRouter()
	router.With(mws.HttpMetrics).Get("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.True(t, mockMetrics.called)
	assert.Equal(t, http.StatusCreated, mockMetrics.code)
	assert.Equal(t, http.MethodGet, mockMetrics.method)
	assert.Equal(t, "/test", mockMetrics.path)
	assert.Greater(t, mockMetrics.latency, 0.0)
}
