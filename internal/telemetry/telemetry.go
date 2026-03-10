package telemetry

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Setup initialises the OTel meter provider with a Prometheus exporter.
// Call Shutdown on the returned provider during graceful shutdown.
func Setup() (*sdkmetric.MeterProvider, error) {
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(mp)

	return mp, nil
}
