package otel

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func OTelMetrics(ctx context.Context, exporter metric.Reader) (http.Handler, func(context.Context) error, error) {
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	return promhttp.Handler(), provider.Shutdown, nil
}

func PrometheusExporter() (*prometheus.Exporter, error) {
	e, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create new Prometheus exporter: %w", err)
	}
	return e, nil
}

func MustCreatePrometheusExporter() *prometheus.Exporter {
	e, err := PrometheusExporter()
	if err != nil {
		panic(err)
	}
	return e
}
