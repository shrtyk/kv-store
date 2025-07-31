package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	p "github.com/prometheus/client_golang/prometheus"
)

type Metrics interface {
	Put()
	Delete()
	Get()
}

type metrics struct {
	putsCounter   p.Counter
	deleteCounter p.Counter
	getCounter    p.Counter
}

func NewPrometheusMetrics() *metrics {
	puts := p.NewCounter(p.CounterOpts{
		Name: "puts_total",
		Help: "The total number of puts",
	})
	dels := p.NewCounter(p.CounterOpts{
		Name: "deletes_total",
		Help: "The total number of deletes",
	})
	gets := p.NewCounter(p.CounterOpts{
		Name: "gets_total",
		Help: "The total number of lookups",
	})
	prometheus.MustRegister(puts, dels, gets)
	return &metrics{
		putsCounter:   puts,
		deleteCounter: dels,
		getCounter:    gets,
	}
}

func (m *metrics) Put() {
	m.putsCounter.Inc()
}

func (m *metrics) Delete() {
	m.deleteCounter.Inc()
}

func (m *metrics) Get() {
	m.getCounter.Inc()
}

type mock struct{}

func NewMockMetrics() *mock {
	return &mock{}
}

func (m *mock) Put()    {}
func (m *mock) Delete() {}
func (m *mock) Get()    {}
