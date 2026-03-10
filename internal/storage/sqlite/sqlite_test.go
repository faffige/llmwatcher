package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/faffige/llmwatcher/internal/provider"
	"github.com/faffige/llmwatcher/internal/storage/sqlite"
)

func TestRecordCall_InsertAndClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	rec := &provider.CallRecord{
		ID:           "01JTEST000000000000000001",
		Provider:     "openai",
		Model:        "gpt-4o-2024-08-06",
		StartedAt:    time.Now().UTC(),
		DurationMs:   150,
		Method:       "POST",
		Path:         "/v1/chat/completions",
		Operation:    "chat",
		StatusCode:   200,
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		RequestBody:  []byte(`{"model":"gpt-4o"}`),
		ResponseBody: []byte(`{"model":"gpt-4o-2024-08-06"}`),
	}

	err = store.RecordCall(context.Background(), rec)
	if err != nil {
		t.Fatalf("RecordCall failed: %v", err)
	}
}

func TestRecordCall_DuplicateID(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	rec := &provider.CallRecord{
		ID:         "01JTEST000000000000000002",
		Provider:   "openai",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		StatusCode: 200,
		StartedAt:  time.Now().UTC(),
	}

	if err := store.RecordCall(context.Background(), rec); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	err = store.RecordCall(context.Background(), rec)
	if err == nil {
		t.Fatal("expected error on duplicate ID, got nil")
	}
}

func TestRecordCall_ErrorFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer store.Close()

	rec := &provider.CallRecord{
		ID:         "01JTEST000000000000000003",
		Provider:   "openai",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		StatusCode: 401,
		StartedAt:  time.Now().UTC(),
		ErrorType:  "invalid_request_error",
		ErrorMsg:   "invalid api key",
	}

	err = store.RecordCall(context.Background(), rec)
	if err != nil {
		t.Fatalf("RecordCall with error fields failed: %v", err)
	}
}
