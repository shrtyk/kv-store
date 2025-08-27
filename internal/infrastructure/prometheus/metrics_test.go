package metrics

import (
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

var m *metrics

func TestMain(m_ *testing.M) {
	// Unregister all collectors before running tests to avoid panics from re-registration.
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	m = NewPrometheusMetrics()
	code := m_.Run()
	os.Exit(code)
}

func TestNewPrometheusMetrics(t *testing.T) {
	assert.NotNil(t, m)
	assert.NotNil(t, m.requests)
	assert.NotNil(t, m.requestsHistogram)
	assert.NotNil(t, m.kvOperationsCounter)
	assert.NotNil(t, m.kvOperationsHistogram)
}

func TestKvOperations(t *testing.T) {
	m.kvOperationsCounter.Reset()
	m.kvOperationsHistogram.Reset()

	// HTTP Put
	m.HttpPut("test_key_put", 0.1)
	metric := &dto.Metric{}
	err := m.kvOperationsCounter.WithLabelValues(string(httpApi), string(putOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist := m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(putOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// HTTP Delete
	m.HttpDelete("test_key_del", 0.1)
	metric.Reset()
	err = m.kvOperationsCounter.WithLabelValues(string(httpApi), string(deleteOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist = m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(deleteOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// HTTP Get
	m.HttpGet("test_key_get", 0.1)
	metric.Reset()
	err = m.kvOperationsCounter.WithLabelValues(string(httpApi), string(getOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist = m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(getOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// GRPC Put
	m.GrpcPut("test_key_put_grpc", 0.1)
	metric.Reset()
	err = m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(putOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist = m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(putOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// GRPC Delete
	m.GrpcDelete("test_key_del_grpc", 0.1)
	metric.Reset()
	err = m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(deleteOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist = m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(deleteOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// GRPC Get
	m.GrpcGet("test_key_get_grpc", 0.1)
	metric.Reset()
	err = m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(getOp)).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())
	metric.Reset()
	hist = m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(getOp))
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())
}

func TestRequestMetrics(t *testing.T) {
	m.requests.Reset()
	m.requestsHistogram.Reset()

	// HTTP Request
	m.HttpRequest(http.StatusOK, http.MethodGet, "/test", 0.1)
	metric := &dto.Metric{}
	httpLabels := []string{string(httpApi), strconv.Itoa(http.StatusOK), http.MethodGet, "/test"}
	err := m.requests.WithLabelValues(httpLabels...).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	hist := m.requestsHistogram.WithLabelValues(httpLabels...)
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	// gRPC Request
	m.GrpcRequest(codes.OK, "TestService", "TestMethod", 0.1)
	grpcLabels := []string{string(grpcApi), codes.OK.String(), "TestMethod", "TestService"}
	err = m.requests.WithLabelValues(grpcLabels...).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	hist = m.requestsHistogram.WithLabelValues(grpcLabels...)
	err = hist.(prometheus.Metric).Write(metric)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())
}

func TestMockMetrics(t *testing.T) {
	mock := NewMockMetrics()
	assert.NotNil(t, mock)

	mock.HttpPut("key", 0.1)
	mock.HttpDelete("key", 0.1)
	mock.HttpGet("key", 0.1)
	mock.GrpcPut("key", 0.1)
	mock.GrpcDelete("key", 0.1)
	mock.GrpcGet("key", 0.1)
	mock.HttpRequest(200, "GET", "/path", 0.1)
	mock.GrpcRequest(codes.OK, "service", "method", 0.1)
}
