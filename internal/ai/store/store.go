// Package store implements the SQLite-backed persistence layer for AI chat:
// conversations, messages, request logs, and tool-call audit records.
//
// The package is split across several files by concern:
//   - store.go          Store type, Open/Close lifecycle
//   - migrate.go        schema creation and incremental migrations
//   - models.go         Conversation, Message, RequestLog value types
//   - conversations.go  conversation CRUD, fork, archive/restore
//   - messages.go       turn lifecycle and message operations
//   - request_logs.go   request-log insertion and retention cleanup
//   - tasks.go          durable agent task queue: leases, checkpoints, recovery
//   - helpers.go        ID generation and small scan/format helpers
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Store is a handle to the AI chat SQLite database.
type Store struct {
	db *sql.DB
}

// Options controls optional Open behavior. ReconcilePending is true for the
// long-running server (which is the only process that should claim abandoned
// pending rows on startup) and false for short-lived secondary processes like
// the bs-ai-chat CLI, which must not cancel another process's in-flight turn.
type Options struct {
	ReconcilePending bool
}

// Open opens (creating if needed) the SQLite database at path, runs migrations,
// and reconciles any messages left pending by a previous unclean shutdown.
// It preserves the legacy default behavior: ReconcilePending is true.
func Open(path string) (*Store, error) {
	return OpenWithOptions(path, Options{ReconcilePending: true})
}

// OpenWithOptions opens the SQLite database with explicit control over the
// pending-message reconciliation step. Callers that share the database with a
// running server (e.g. the bs-ai-chat CLI) must pass ReconcilePending=false
// so they do not cancel another process's in-flight assistant row.
func OpenWithOptions(path string, opts Options) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(path) + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		db.Close()
		return nil, fmt.Errorf("verify sqlite foreign_keys: value=%d err=%v", foreignKeys, err)
	}
	if opts.ReconcilePending {
		if _, err := db.Exec(`UPDATE messages SET status = 'cancelled' WHERE status = 'pending'`); err != nil {
			db.Close()
			return nil, fmt.Errorf("reconcile pending messages: %w", err)
		}
	}
	return store, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
