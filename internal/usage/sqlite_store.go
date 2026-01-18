package usage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore manages SQLite-based persistence for usage statistics.
type SQLiteStore struct {
	db   *sql.DB
	path string
}

const schema = `
CREATE TABLE IF NOT EXISTS usage_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key TEXT NOT NULL,
    model TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    auth_index TEXT NOT NULL DEFAULT '',
    failed INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    dedup_key TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_dedup
    ON usage_records(dedup_key);

CREATE INDEX IF NOT EXISTS idx_usage_records_lookup
    ON usage_records(api_key, model, timestamp);
`

// NewSQLiteStore creates a new SQLite store for usage statistics.
// If path is empty, it defaults to <auth-dir>/usage.db.
//
// Parameters:
//   - path: The path to the SQLite database file
//
// Returns:
//   - *SQLiteStore: A new SQLite store instance
//   - error: An error if the database could not be opened
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	// Expand tilde in path
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to expand home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	return &SQLiteStore{
		db:   db,
		path: path,
	}, nil
}

// EnsureSchema creates the database schema if it doesn't exist.
//
// Parameters:
//   - ctx: The context for the operation
//
// Returns:
//   - error: An error if the schema could not be created
func (s *SQLiteStore) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store not initialized")
	}

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// InsertRecord persists a single usage record to the database.
// Duplicates (based on dedup_key) are silently ignored.
//
// Parameters:
//   - ctx: The context for the operation
//   - apiKey: The API key identifier
//   - model: The model name
//   - detail: The request detail to persist
//
// Returns:
//   - error: An error if the record could not be inserted
func (s *SQLiteStore) InsertRecord(ctx context.Context, apiKey, model string, detail RequestDetail) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store not initialized")
	}

	key := dedupKey(apiKey, model, detail)
	timestamp := detail.Timestamp.UTC().Format(time.RFC3339Nano)
	failed := 0
	if detail.Failed {
		failed = 1
	}

	query := `
		INSERT OR IGNORE INTO usage_records (
			api_key, model, timestamp, source, auth_index, failed,
			input_tokens, output_tokens, reasoning_tokens, cached_tokens, total_tokens,
			dedup_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		apiKey, model, timestamp, detail.Source, detail.AuthIndex, failed,
		detail.Tokens.InputTokens, detail.Tokens.OutputTokens,
		detail.Tokens.ReasoningTokens, detail.Tokens.CachedTokens,
		detail.Tokens.TotalTokens, key,
	)

	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}

	return nil
}

// LoadAll retrieves all usage records from the database and returns them as a StatisticsSnapshot.
//
// Parameters:
//   - ctx: The context for the operation
//
// Returns:
//   - StatisticsSnapshot: A snapshot containing all records from the database
//   - error: An error if the records could not be loaded
func (s *SQLiteStore) LoadAll(ctx context.Context) (StatisticsSnapshot, error) {
	snapshot := StatisticsSnapshot{
		APIs: make(map[string]APISnapshot),
	}

	if s == nil || s.db == nil {
		return snapshot, fmt.Errorf("store not initialized")
	}

	query := `
		SELECT api_key, model, timestamp, source, auth_index, failed,
		       input_tokens, output_tokens, reasoning_tokens, cached_tokens, total_tokens
		FROM usage_records
		ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return snapshot, fmt.Errorf("failed to query records: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var apiKey, model, timestampStr, source, authIndex string
		var failed int
		var inputTokens, outputTokens, reasoningTokens, cachedTokens, totalTokens int64

		err := rows.Scan(
			&apiKey, &model, &timestampStr, &source, &authIndex, &failed,
			&inputTokens, &outputTokens, &reasoningTokens, &cachedTokens, &totalTokens,
		)
		if err != nil {
			return snapshot, fmt.Errorf("failed to scan record: %w", err)
		}

		timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
		if err != nil {
			// Try fallback to RFC3339
			timestamp, err = time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				continue // Skip malformed timestamps
			}
		}

		detail := RequestDetail{
			Timestamp: timestamp,
			Source:    source,
			AuthIndex: authIndex,
			Tokens: TokenStats{
				InputTokens:     inputTokens,
				OutputTokens:    outputTokens,
				ReasoningTokens: reasoningTokens,
				CachedTokens:    cachedTokens,
				TotalTokens:     totalTokens,
			},
			Failed: failed != 0,
		}

		// Build snapshot structure
		apiSnapshot, ok := snapshot.APIs[apiKey]
		if !ok {
			apiSnapshot = APISnapshot{
				Models: make(map[string]ModelSnapshot),
			}
		}

		modelSnapshot, ok := apiSnapshot.Models[model]
		if !ok {
			modelSnapshot = ModelSnapshot{
				Details: []RequestDetail{},
			}
		}

		modelSnapshot.Details = append(modelSnapshot.Details, detail)
		apiSnapshot.Models[model] = modelSnapshot
		snapshot.APIs[apiKey] = apiSnapshot
	}

	if err := rows.Err(); err != nil {
		return snapshot, fmt.Errorf("error iterating records: %w", err)
	}

	return snapshot, nil
}

// PersistSnapshot saves all records from a StatisticsSnapshot to the database.
// Uses the same deduplication logic as in-memory merge (skips existing records).
//
// Parameters:
//   - ctx: The context for the operation
//   - snapshot: The snapshot to persist
//
// Returns:
//   - added: Number of records added
//   - skipped: Number of records skipped (duplicates)
//   - error: An error if the operation failed
func (s *SQLiteStore) PersistSnapshot(ctx context.Context, snapshot StatisticsSnapshot) (added, skipped int, err error) {
	if s == nil || s.db == nil {
		return 0, 0, fmt.Errorf("store not initialized")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO usage_records (
			api_key, model, timestamp, source, auth_index, failed,
			input_tokens, output_tokens, reasoning_tokens, cached_tokens, total_tokens,
			dedup_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for apiKey, apiSnapshot := range snapshot.APIs {
		for model, modelSnapshot := range apiSnapshot.Models {
			for _, detail := range modelSnapshot.Details {
				key := dedupKey(apiKey, model, detail)
				timestamp := detail.Timestamp.UTC().Format(time.RFC3339Nano)
				failed := 0
				if detail.Failed {
					failed = 1
				}

				result, err := stmt.ExecContext(ctx,
					apiKey, model, timestamp, detail.Source, detail.AuthIndex, failed,
					detail.Tokens.InputTokens, detail.Tokens.OutputTokens,
					detail.Tokens.ReasoningTokens, detail.Tokens.CachedTokens,
					detail.Tokens.TotalTokens, key,
				)
				if err != nil {
					return added, skipped, fmt.Errorf("failed to insert record: %w", err)
				}

				rows, _ := result.RowsAffected()
				if rows > 0 {
					added++
				} else {
					skipped++
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return added, skipped, nil
}

// Close closes the database connection.
//
// Returns:
//   - error: An error if the connection could not be closed
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
