package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	tutils "github.com/shrtyk/kv-store/internal/tests/testutils"
	"github.com/shrtyk/kv-store/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

type mockMetrics struct {
	called  bool
	code    int
	method  string
	path    string
	latency float64
}

func (m *mockMetrics) GrpcPut(key string, duration float64)    {}
func (m *mockMetrics) GrpcDelete(key string, duration float64) {}
func (m *mockMetrics) GrpcGet(key string, duration float64)    {}

func (m *mockMetrics) HttpRequest(code int, method, path string, latency float64) {
	m.called = true
	m.code = code
	m.method = method
	m.path = path
	m.latency = latency
}
func (m *mockMetrics) GrpcRequest(code codes.Code, service, method string, latency float64) {}
func (m *mockMetrics) HttpPut(key string, duration float64)                                 {}
func (m *mockMetrics) HttpDelete(key string, duration float64)                              {}
func (m *mockMetrics) HttpGet(key string, duration float64)                                 {}

func TestHttpMetrics(t *testing.T) {
	l, _ := tutils.NewMockLogger()
	mockMetrics := &mockMetrics{}
	mws := NewMiddlewares(l, mockMetrics)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("test"))
		require.NoError(t, err)
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

func TestLogging(t *testing.T) {
	l, buf := tutils.NewMockLogger()
	mws := NewMiddlewares(l, &mockMetrics{})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.FromCtx(r.Context())
		logger.Info("test message")
	})

	router := chi.NewRouter()
	router.With(mws.Logging).Get("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Contains(t, buf.String(), "test message")
	assert.Contains(t, buf.String(), "ip")
	assert.Contains(t, buf.String(), "user-agent")
	assert.Contains(t, buf.String(), "request_id")
	assert.Contains(t, buf.String(), "method")
	assert.Contains(t, buf.String(), "url")
}
