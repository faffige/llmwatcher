package telemetry

import (
	"context"
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/faffige/llmwatcher/internal/provider"
)

const meterName = "github.com/faffige/llmwatcher"

// Metrics holds the OTel metric instruments for LLM observability.
type Metrics struct {
	requestsTotal   metric.Int64Counter
	tokensTotal     metric.Int64Counter
	durationSeconds metric.Float64Histogram
}

// NewMetrics creates and registers the LLM metric instruments.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(meterName)

	requests, err := meter.Int64Counter("llmwatcher.requests.total",
		metric.WithDescription("Total number of LLM API requests proxied"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating requests counter: %w", err)
	}

	tokens, err := meter.Int64Counter("llmwatcher.tokens.total",
		metric.WithDescription("Total number of tokens consumed"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating tokens counter: %w", err)
	}

	duration, err := meter.Float64Histogram("llmwatcher.request.duration.seconds",
		metric.WithDescription("Duration of LLM API requests in seconds"),
		metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60),
	)
	if err != nil {
		return nil, fmt.Errorf("creating duration histogram: %w", err)
	}

	return &Metrics{
		requestsTotal:   requests,
		tokensTotal:     tokens,
		durationSeconds: duration,
	}, nil
}

// Record updates all metrics from a CallRecord.
func (m *Metrics) Record(ctx context.Context, rec *provider.CallRecord) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", rec.Provider),
		attribute.String("model", rec.Model),
		attribute.String("status", strconv.Itoa(rec.StatusCode)),
		attribute.Bool("stream", rec.IsStream),
	}
	attrSet := metric.WithAttributes(attrs...)

	m.requestsTotal.Add(ctx, 1, attrSet)

	m.durationSeconds.Record(ctx, float64(rec.DurationMs)/1000.0, attrSet)

	if rec.InputTokens > 0 {
		m.tokensTotal.Add(ctx, int64(rec.InputTokens), metric.WithAttributes(
			append(attrs, attribute.String("direction", "input"))...,
		))
	}
	if rec.OutputTokens > 0 {
		m.tokensTotal.Add(ctx, int64(rec.OutputTokens), metric.WithAttributes(
			append(attrs, attribute.String("direction", "output"))...,
		))
	}
}
