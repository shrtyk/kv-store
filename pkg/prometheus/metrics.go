package metrics

import (
	"strconv"

	p "github.com/prometheus/client_golang/prometheus"
)

type Metrics interface {
	HttpRequest(code int, method, path string, latency float64)
	Put(key string, duration float64)
	Delete(key string, duration float64)
	Get(key string, duration float64)
}

type metrics struct {
	httpRequests          *p.CounterVec
	httpRequestsHistogram p.Histogram

	putsCounter   *p.CounterVec
	deleteCounter *p.CounterVec
	getCounter    *p.CounterVec
	putHistogram  p.Histogram
	delHistogram  p.Histogram
	getHistogram  p.Histogram
}

func NewPrometheusMetrics() *metrics {
	keyLabel := []string{"key"}
	put := p.NewCounterVec(p.CounterOpts{
		Name: "http_puts_total",
		Help: "The total number of http puts",
	}, keyLabel)
	del := p.NewCounterVec(p.CounterOpts{
		Name: "http_deletes_total",
		Help: "The total number of http deletes",
	}, keyLabel)
	get := p.NewCounterVec(p.CounterOpts{
		Name: "http_gets_total",
		Help: "The total number of http lookups",
	}, keyLabel)
	putHist := p.NewHistogram(p.HistogramOpts{
		Name:    "puts_latency_seconds",
		Help:    "The latency of puts in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	delHist := p.NewHistogram(p.HistogramOpts{
		Name:    "deletes_latency_seconds",
		Help:    "The latency of deletes in seconds",
		Buckets: p.LinearBuckets(0.0001, 0.0001, 10),
	})
	getHist := p.NewHistogram(p.HistogramOpts{
		Name:    "gets_latency_seconds",
		Help:    "The latency of gets in seconds",
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
		put, del, get, putHist,
		delHist, getHist, httpRequests, httpReqHist)
	return &metrics{
		putsCounter:   put,
		deleteCounter: del,
		getCounter:    get,
		putHistogram:  putHist,
		delHistogram:  delHist,
		getHistogram:  getHist,

		httpRequests:          httpRequests,
		httpRequestsHistogram: httpReqHist,
	}
}

func (m *metrics) Put(key string, duration float64) {
	m.putsCounter.WithLabelValues(key).Inc()
	m.putHistogram.Observe(duration)
}

func (m *metrics) Delete(key string, duration float64) {
	m.deleteCounter.WithLabelValues(key).Inc()
	m.delHistogram.Observe(duration)
}

func (m *metrics) Get(key string, duration float64) {
	m.getCounter.WithLabelValues(key).Inc()
	m.getHistogram.Observe(duration)
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

func (m *mock) Put(key string, duration float64)                           {}
func (m *mock) Delete(key string, duration float64)                        {}
func (m *mock) Get(key string, duration float64)                           {}
func (m *mock) HttpRequest(code int, method, path string, latency float64) {}
