package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/faffige/llmwatcher/internal/provider"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS call_records (
	id            TEXT PRIMARY KEY,
	provider      TEXT NOT NULL,
	model         TEXT NOT NULL DEFAULT '',
	started_at    DATETIME NOT NULL,
	duration_ms   INTEGER NOT NULL DEFAULT 0,
	method        TEXT NOT NULL,
	path          TEXT NOT NULL,
	operation     TEXT NOT NULL DEFAULT '',
	status_code   INTEGER NOT NULL,
	is_stream     INTEGER NOT NULL DEFAULT 0,
	input_tokens  INTEGER NOT NULL DEFAULT 0,
	output_tokens INTEGER NOT NULL DEFAULT 0,
	total_tokens  INTEGER NOT NULL DEFAULT 0,
	error_type    TEXT NOT NULL DEFAULT '',
	error_msg     TEXT NOT NULL DEFAULT '',
	request_body  BLOB,
	response_body BLOB
);

CREATE INDEX IF NOT EXISTS idx_call_records_started_at ON call_records(started_at);
CREATE INDEX IF NOT EXISTS idx_call_records_provider ON call_records(provider);
CREATE INDEX IF NOT EXISTS idx_call_records_model ON call_records(model);
`

// Store implements storage.Store backed by SQLite.
type Store struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at the given path and initialises the schema.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	// SQLite pragmas for performance.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialising schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) RecordCall(ctx context.Context, rec *provider.CallRecord) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO call_records (
			id, provider, model, started_at, duration_ms,
			method, path, operation, status_code, is_stream,
			input_tokens, output_tokens, total_tokens,
			error_type, error_msg, request_body, response_body
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.ID, rec.Provider, rec.Model, rec.StartedAt, rec.DurationMs,
		rec.Method, rec.Path, rec.Operation, rec.StatusCode, rec.IsStream,
		rec.InputTokens, rec.OutputTokens, rec.TotalTokens,
		rec.ErrorType, rec.ErrorMsg, rec.RequestBody, rec.ResponseBody,
	)
	if err != nil {
		return fmt.Errorf("inserting call record: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
