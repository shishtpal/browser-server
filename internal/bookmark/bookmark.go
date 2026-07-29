// Package bookmark holds the bookmark domain logic shared by the REST API
// handlers in internal/handlers, the bookmark importer, and the AI tools in
// internal/ai/tools.
//
// The column list, row scanner, capture-idempotent insert, and search query
// used to be written out separately in each of those places. They now live
// here so the bookmark model is defined once.
package bookmark

import (
	"database/sql"
	"errors"
)

// Columns is the canonical column list for bookmark SELECT queries.
const Columns = "id, user_id, title, url, description, tags, folder_path, created_at, updated_at"

// Field limits shared by bookmark API consumers.
const (
	MaxTitleLength       = 200
	MaxDescriptionLength = 2000
	MaxURLLength         = 2048
	MaxFolderPathLength  = 500
	MaxTagLength         = 100
)

var (
	// ErrNotFound is returned by bookmark lookups when no row exists.
	ErrNotFound = sql.ErrNoRows
	// ErrBookmarkNotFound is returned when an update matches no bookmark.
	ErrBookmarkNotFound = errors.New("bookmark not found")
)
