package storage

import (
	"context"

	"github.com/faffige/llmwatcher/internal/provider"
)

// Store persists and queries CallRecords.
type Store interface {
	// RecordCall inserts a single call record.
	RecordCall(ctx context.Context, rec *provider.CallRecord) error

	// Close releases any resources held by the store.
	Close() error
}
