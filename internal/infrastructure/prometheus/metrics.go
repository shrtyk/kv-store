package metrics

import (
	"strconv"

	p "github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	httpRequests          *p.CounterVec
	httpRequestsHistogram p.Histogram

	putsCounter   *p.CounterVec
	deleteCounter *p.CounterVec
	getCounter    *p.CounterVec
	putHistogram  p.Histogram
	delHistogram  p.Histogram
	getHistogram  p.Histogram

	grpcPutsCounter   *p.CounterVec
	grpcDeleteCounter *p.CounterVec
	grpcGetCounter    *p.CounterVec
	grpcPutHistogram  p.Histogram
	grpcDelHistogram  p.Histogram
	grpcGetHistogram  p.Histogram
}

func NewPrometheusMetrics() *metrics {
	keyLabel := []string{"key"}
	httpPut := p.NewCounterVec(p.CounterOpts{
		Name: "http_puts_total",
		Help: "The total number of http puts",
	}, keyLabel)
	httpDel := p.NewCounterVec(p.CounterOpts{
		Name: "http_deletes_total",
		Help: "The total number of http deletes",
	}, keyLabel)
	httpGet := p.NewCounterVec(p.CounterOpts{
		Name: "http_gets_total",
		Help: "The total number of http lookups",
	}, keyLabel)
	httpPutHist := p.NewHistogram(p.HistogramOpts{
		Name:    "http_puts_latency_seconds",
		Help:    "The latency of puts in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	httpDelHist := p.NewHistogram(p.HistogramOpts{
		Name:    "http_deletes_latency_seconds",
		Help:    "The latency of deletes in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	httpGetHist := p.NewHistogram(p.HistogramOpts{
		Name:    "http_gets_latency_seconds",
		Help:    "The latency of gets in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})

	grpcPut := p.NewCounterVec(p.CounterOpts{
		Name: "grpc_puts_total",
		Help: "The total number of grpc puts",
	}, keyLabel)
	grpcDel := p.NewCounterVec(p.CounterOpts{
		Name: "grpc_deletes_total",
		Help: "The total number of grpc deletes",
	}, keyLabel)
	grpcGet := p.NewCounterVec(p.CounterOpts{
		Name: "grpc_gets_total",
		Help: "The total number of grpc lookups",
	}, keyLabel)
	grpcPutHist := p.NewHistogram(p.HistogramOpts{
		Name:    "grpc_puts_latency_seconds",
		Help:    "The latency of grpc puts in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	grpcDelHist := p.NewHistogram(p.HistogramOpts{
		Name:    "grpc_deletes_latency_seconds",
		Help:    "The latency of grpc deletes in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	grpcGetHist := p.NewHistogram(p.HistogramOpts{
		Name:    "grpc_gets_latency_seconds",
		Help:    "The latency of grpc gets in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})

	httpRequests := p.NewCounterVec(p.CounterOpts{
		Name: "http_requests_total",
		Help: "The total number of http requests",
	}, []string{"code", "method", "path"})
	httpReqHist := p.NewHistogram(p.HistogramOpts{
		Name:    "http_requests_seconds",
		Help:    "The http requests latency in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})

	p.MustRegister(
		httpPut, httpDel, httpGet, httpPutHist,
		httpDelHist, httpGetHist, httpRequests, httpReqHist,
		grpcPut, grpcDel, grpcGet, grpcPutHist,
		grpcDelHist, grpcGetHist,
	)
	return &metrics{
		putsCounter:   httpPut,
		deleteCounter: httpDel,
		getCounter:    httpGet,
		putHistogram:  httpPutHist,
		delHistogram:  httpDelHist,
		getHistogram:  httpGetHist,

		grpcPutsCounter:   grpcPut,
		grpcDeleteCounter: grpcDel,
		grpcGetCounter:    grpcGet,
		grpcPutHistogram:  grpcPutHist,
		grpcDelHistogram:  grpcDelHist,
		grpcGetHistogram:  grpcGetHist,

		httpRequests:          httpRequests,
		httpRequestsHistogram: httpReqHist,
	}
}

func (m *metrics) HttpPut(key string, duration float64) {
	m.putsCounter.WithLabelValues(key).Inc()
	m.putHistogram.Observe(duration)
}

func (m *metrics) HttpDelete(key string, duration float64) {
	m.deleteCounter.WithLabelValues(key).Inc()
	m.delHistogram.Observe(duration)
}

func (m *metrics) HttpGet(key string, duration float64) {
	m.getCounter.WithLabelValues(key).Inc()
	m.getHistogram.Observe(duration)
}

func (m *metrics) GrpcPut(key string, duration float64) {
	m.grpcPutsCounter.WithLabelValues(key).Inc()
	m.grpcPutHistogram.Observe(duration)
}

func (m *metrics) GrpcDelete(key string, duration float64) {
	m.grpcDeleteCounter.WithLabelValues(key).Inc()
	m.grpcDelHistogram.Observe(duration)
}

func (m *metrics) GrpcGet(key string, duration float64) {
	m.grpcGetCounter.WithLabelValues(key).Inc()
	m.grpcGetHistogram.Observe(duration)
}

func (m *metrics) HttpRequest(code int, method, path string, latency float64) {
	m.httpRequests.WithLabelValues(
		strconv.Itoa(code),
		method,
		path,
	).Inc()
	m.httpRequestsHistogram.Observe(latency)
}

type mock struct{}

func NewMockMetrics() *mock {
	return &mock{}
}

func (m *mock) HttpPut(key string, duration float64)                       {}
func (m *mock) HttpDelete(key string, duration float64)                    {}
func (m *mock) HttpGet(key string, duration float64)                       {}
func (m *mock) HttpRequest(code int, method, path string, latency float64) {}
func (m *mock) GrpcPut(key string, duration float64)                       {}
func (m *mock) GrpcDelete(key string, duration float64)                    {}
func (m *mock) GrpcGet(key string, duration float64)                       {}
