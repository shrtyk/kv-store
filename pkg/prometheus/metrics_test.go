package metrics

import (
	"net/http"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

var m *metrics

func TestMain(m_ *testing.M) {
	m = NewPrometheusMetrics()
	code := m_.Run()
	os.Exit(code)
}

func TestNewPrometheusMetrics(t *testing.T) {
	assert.NotNil(t, m)
	assert.NotNil(t, m.httpRequests)
	assert.NotNil(t, m.httpRequestsHistogram)
	assert.NotNil(t, m.putsCounter)
	assert.NotNil(t, m.deleteCounter)
	assert.NotNil(t, m.getCounter)
	assert.NotNil(t, m.putHistogram)
	assert.NotNil(t, m.delHistogram)
	assert.NotNil(t, m.getHistogram)
}

func TestMetricsPut(t *testing.T) {
	m.Put("test_key", 0.1)

	metric := &dto.Metric{}
	m.putsCounter.WithLabelValues("test_key").Write(metric)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	m.putHistogram.Write(metric)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	prometheus.Unregister(m.putsCounter)
}

func TestMetricsDelete(t *testing.T) {
	m.Delete("test_key", 0.1)

	metric := &dto.Metric{}
	m.deleteCounter.WithLabelValues("test_key").Write(metric)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	m.delHistogram.Write(metric)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	prometheus.Unregister(m.deleteCounter)
}

func TestMetricsGet(t *testing.T) {
	m.Get("test_key", 0.1)

	metric := &dto.Metric{}
	m.getCounter.WithLabelValues("test_key").Write(metric)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	m.getHistogram.Write(metric)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	prometheus.Unregister(m.getCounter)
}

func TestMetricsHttpRequest(t *testing.T) {
	m.HttpRequest(http.StatusOK, http.MethodGet, "/test", 0.1)

	metric := &dto.Metric{}
	m.httpRequests.WithLabelValues("200", "GET", "/test").Write(metric)
	assert.Equal(t, float64(1), metric.Counter.GetValue())

	metric.Reset()
	m.httpRequestsHistogram.Write(metric)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())

	prometheus.Unregister(m.httpRequests)
}

func TestMockMetrics(t *testing.T) {
	mock := NewMockMetrics()
	assert.NotNil(t, mock)

	mock.Put("key", 0.1)
	mock.Delete("key", 0.1)
	mock.Get("key", 0.1)
	mock.HttpRequest(200, "GET", "/path", 0.1)
}
