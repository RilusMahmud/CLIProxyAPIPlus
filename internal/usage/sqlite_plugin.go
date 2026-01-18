package usage

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

// SQLitePlugin implements coreusage.Plugin to persist usage records to SQLite.
type SQLitePlugin struct {
	store *SQLiteStore
	stats *RequestStatistics
}

// NewSQLitePlugin creates a new SQLite plugin that persists usage records.
//
// Parameters:
//   - store: The SQLite store to persist records to
//   - stats: The in-memory statistics store (used to get API identifier context)
//
// Returns:
//   - *SQLitePlugin: A new SQLite plugin instance
func NewSQLitePlugin(store *SQLiteStore, stats *RequestStatistics) *SQLitePlugin {
	return &SQLitePlugin{
		store: store,
		stats: stats,
	}
}

// HandleUsage implements coreusage.Plugin.
// It persists each usage record to SQLite for long-term storage.
//
// Parameters:
//   - ctx: The context for the usage record
//   - record: The usage record to persist
func (p *SQLitePlugin) HandleUsage(ctx context.Context, record coreusage.Record) {
	if p == nil || p.store == nil {
		return
	}

	if !statisticsEnabled.Load() {
		return
	}

	// Convert record to RequestDetail
	timestamp := record.RequestedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	detail := normaliseDetail(record.Detail)

	// Resolve API key identifier
	apiKey := record.APIKey
	if apiKey == "" {
		apiKey = resolveAPIIdentifier(ctx, record)
	}

	// Determine if request failed
	failed := record.Failed
	if !failed {
		failed = !resolveSuccess(ctx)
	}

	// Get model name
	modelName := record.Model
	if modelName == "" {
		modelName = "unknown"
	}

	// Create request detail
	requestDetail := RequestDetail{
		Timestamp: timestamp,
		Source:    record.Source,
		AuthIndex: record.AuthIndex,
		Tokens:    detail,
		Failed:    failed,
	}

	// Persist to SQLite (async, don't block the handler)
	// We use a background context to avoid cancellation propagation
	go func() {
		bgCtx := context.Background()
		if err := p.store.InsertRecord(bgCtx, apiKey, modelName, requestDetail); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"api_key": apiKey,
				"model":   modelName,
			}).Error("failed to persist usage record to SQLite")
		}
	}()
}

// LoadAndMerge loads all records from SQLite and merges them into the in-memory statistics.
// This is called on startup to restore persisted data.
//
// Parameters:
//   - ctx: The context for the operation
//
// Returns:
//   - error: An error if the operation failed
func (p *SQLitePlugin) LoadAndMerge(ctx context.Context) error {
	if p == nil || p.store == nil || p.stats == nil {
		return nil
	}

	snapshot, err := p.store.LoadAll(ctx)
	if err != nil {
		return err
	}

	result := p.stats.MergeSnapshot(snapshot)
	log.WithFields(log.Fields{
		"added":   result.Added,
		"skipped": result.Skipped,
	}).Info("restored usage statistics from SQLite")

	return nil
}
