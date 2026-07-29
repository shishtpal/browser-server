// Package history holds the browsing-history domain logic shared by the REST
// API handlers in internal/handlers, the history importer, the omnibox search,
// and the AI tools in internal/ai/tools.
//
// The SQLite timestamp parsing, column list, row scanner, insert statement,
// and aggregate queries used to be repeated across those callers; they now
// live here so the history model is defined once, mirroring the layout of the
// sibling bookmark and prompt packages.
package history

import (
	"database/sql"
	"strings"
)

// Columns is the canonical column list for history SELECT queries. It matches
// the fields of models.History one-to-one, so every query that scans a row
// must select these columns and use Scan.
const Columns = "id, user_id, url, title, domain, visited_at, duration"

// ErrNotFound is returned by history lookups when no row exists.
var ErrNotFound = sql.ErrNoRows

// SearchTerms builds the WHERE fragment matching every whitespace-separated
// term in a search string against the given columns. Each term must match
// (AND), while the columns are alternatives (OR). It is pure so callers can
// unit-test term logic without a database.
func SearchTerms(search string, columns ...string) (string, []any) {
	var (
		clause strings.Builder
		args   []any
	)
	for _, term := range strings.Fields(search) {
		like := "%" + term + "%"
		clause.WriteString(" AND (")
		for i, column := range columns {
			if i > 0 {
				clause.WriteString(" OR ")
			}
			clause.WriteString(column)
			clause.WriteString(" LIKE ?")
			args = append(args, like)
		}
		clause.WriteString(")")
	}
	return clause.String(), args
}
