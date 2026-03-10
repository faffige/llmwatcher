package pipeline

import (
	"context"
	"crypto/rand"
	"log/slog"
	"sync"

	"github.com/faffige/llmwatcher/internal/provider"
	"github.com/faffige/llmwatcher/internal/storage"
	"github.com/faffige/llmwatcher/internal/telemetry"
	"github.com/oklog/ulid/v2"
)

// Pipeline receives CallRecords on a buffered channel and writes them
// to the store via a single background worker.
type Pipeline struct {
	ch      chan *provider.CallRecord
	store   storage.Store
	metrics *telemetry.Metrics
	logger  *slog.Logger
	wg      sync.WaitGroup
}

// New creates a pipeline with the given buffer size and starts the worker.
// metrics may be nil if telemetry is not configured.
func New(store storage.Store, metrics *telemetry.Metrics, bufSize int, logger *slog.Logger) *Pipeline {
	if bufSize <= 0 {
		bufSize = 256
	}

	p := &Pipeline{
		ch:      make(chan *provider.CallRecord, bufSize),
		store:   store,
		metrics: metrics,
		logger:  logger,
	}

	p.wg.Add(1)
	go p.worker()

	return p
}

// Submit sends a CallRecord to the pipeline for async processing.
// If the buffer is full, the record is dropped and a warning is logged.
func (p *Pipeline) Submit(rec *provider.CallRecord) {
	// Assign a ULID before sending.
	rec.ID = ulid.MustNew(ulid.Now(), rand.Reader).String()

	select {
	case p.ch <- rec:
	default:
		p.logger.Warn("pipeline buffer full, dropping record",
			"provider", rec.Provider,
			"model", rec.Model,
		)
	}
}

// Close signals the worker to drain remaining records and stop.
func (p *Pipeline) Close() {
	close(p.ch)
	p.wg.Wait()
}

func (p *Pipeline) worker() {
	defer p.wg.Done()

	for rec := range p.ch {
		if err := p.store.RecordCall(context.Background(), rec); err != nil {
			p.logger.Error("failed to write call record",
				"id", rec.ID,
				"error", err,
			)
		}

		if p.metrics != nil {
			p.metrics.Record(context.Background(), rec)
		}
	}
}
