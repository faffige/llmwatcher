package pipeline_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/faffige/llmwatcher/internal/pipeline"
	"github.com/faffige/llmwatcher/internal/provider"
)

// mockStore captures records in memory for testing.
type mockStore struct {
	mu      sync.Mutex
	records []*provider.CallRecord
}

func (m *mockStore) RecordCall(_ context.Context, rec *provider.CallRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, rec)
	return nil
}

func (m *mockStore) Close() error { return nil }

func (m *mockStore) len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

func TestPipeline_SubmitAndClose(t *testing.T) {
	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pl := pipeline.New(store, nil, 16, logger)

	pl.Submit(&provider.CallRecord{
		Provider:   "openai",
		Model:      "gpt-4o",
		StartedAt:  time.Now(),
		DurationMs: 100,
	})

	pl.Submit(&provider.CallRecord{
		Provider:   "openai",
		Model:      "gpt-4o-mini",
		StartedAt:  time.Now(),
		DurationMs: 50,
	})

	// Close waits for all records to be processed.
	pl.Close()

	if store.len() != 2 {
		t.Fatalf("expected 2 records, got %d", store.len())
	}

	// Verify ULIDs were assigned.
	for i, rec := range store.records {
		if rec.ID == "" {
			t.Errorf("record %d has empty ID", i)
		}
	}
}

func TestPipeline_AssignsUniqueIDs(t *testing.T) {
	store := &mockStore{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pl := pipeline.New(store, nil, 16, logger)

	for i := 0; i < 10; i++ {
		pl.Submit(&provider.CallRecord{
			Provider:  "openai",
			StartedAt: time.Now(),
		})
	}

	pl.Close()

	ids := make(map[string]bool)
	for _, rec := range store.records {
		if ids[rec.ID] {
			t.Errorf("duplicate ID: %s", rec.ID)
		}
		ids[rec.ID] = true
	}
}
