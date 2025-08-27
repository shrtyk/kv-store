package metrics

import (
	"strconv"

	p "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
)

type apiType string

const (
	httpApi apiType = "http"
	grpcApi apiType = "grpc"
)

type opType string

const (
	getOp    opType = "get"
	putOp    opType = "put"
	deleteOp opType = "delete"
)

type metrics struct {
	requests          *p.CounterVec
	requestsHistogram *p.HistogramVec

	kvOperationsCounter   *p.CounterVec
	kvOperationsHistogram *p.HistogramVec
}

func NewPrometheusMetrics() *metrics {
	opsCounter := p.NewCounterVec(p.CounterOpts{
		Name: "kv_operations_total",
		Help: "The total number of kv operations",
	}, []string{"transport", "operation"})

	opsHistogram := p.NewHistogramVec(p.HistogramOpts{
		Name:    "kv_operations_latency_seconds",
		Help:    "The latency of kv operations in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	}, []string{"transport", "operation"})

	requests := p.NewCounterVec(p.CounterOpts{
		Name: "requests_total",
		Help: "The total number of http/grpc requests",
	}, []string{"transport", "code", "method", "endpoint"})

	requestsHistogram := p.NewHistogramVec(p.HistogramOpts{
		Name:    "requests_seconds",
		Help:    "The http/grpc requests latency in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	}, []string{"transport", "code", "method", "endpoint"})

	p.MustRegister(
		opsCounter, opsHistogram,
		requests, requestsHistogram,
	)

	return &metrics{
		kvOperationsCounter:   opsCounter,
		kvOperationsHistogram: opsHistogram,
		requests:              requests,
		requestsHistogram:     requestsHistogram,
	}
}

// HttpPut increments the http put counter and observes the duration
func (m *metrics) HttpPut(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(httpApi), string(putOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(putOp)).Observe(duration)
}

// HttpDelete increments the http delete counter and observes the duration
func (m *metrics) HttpDelete(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(httpApi), string(deleteOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(deleteOp)).Observe(duration)
}

// HttpGet increments the http get counter and observes the duration
func (m *metrics) HttpGet(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(httpApi), string(getOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(httpApi), string(getOp)).Observe(duration)
}

// GrpcPut increments the grpc put counter and observes the duration
func (m *metrics) GrpcPut(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(putOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(putOp)).Observe(duration)
}

// GrpcDelete increments the grpc delete counter and observes the duration
func (m *metrics) GrpcDelete(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(deleteOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(deleteOp)).Observe(duration)
}

// GrpcGet increments the grpc get counter and observes the duration
func (m *metrics) GrpcGet(key string, duration float64) {
	m.kvOperationsCounter.WithLabelValues(string(grpcApi), string(getOp)).Inc()
	m.kvOperationsHistogram.WithLabelValues(string(grpcApi), string(getOp)).Observe(duration)
}

// HttpRequest increments the request counter and observes the latency
func (m *metrics) HttpRequest(code int, method, path string, latency float64) {
	labels := []string{string(httpApi), strconv.Itoa(code), method, path}
	m.requests.WithLabelValues(labels...).Inc()
	m.requestsHistogram.WithLabelValues(labels...).Observe(latency)
}

// GrpcRequest increments the request counter and observes the latency
func (m *metrics) GrpcRequest(code codes.Code, service, method string, latency float64) {
	labels := []string{string(grpcApi), code.String(), method, service}
	m.requests.WithLabelValues(labels...).Inc()
	m.requestsHistogram.WithLabelValues(labels...).Observe(latency)
}

type mock struct{}

func NewMockMetrics() *mock {
	return &mock{}
}

func (m *mock) HttpPut(key string, duration float64)                                 {}
func (m *mock) HttpDelete(key string, duration float64)                              {}
func (m *mock) HttpGet(key string, duration float64)                                 {}
func (m *mock) HttpRequest(code int, method, path string, latency float64)           {}
func (m *mock) GrpcPut(key string, duration float64)                                 {}
func (m *mock) GrpcDelete(key string, duration float64)                              {}
func (m *mock) GrpcGet(key string, duration float64)                                 {}
func (m *mock) GrpcRequest(code codes.Code, service, method string, latency float64) {}
