package history

import (
	"database/sql"
	"time"
)

// sqliteTimeFormats mirrors the layouts go-sqlite3 uses to (de)serialize
// DATETIME values. Aggregates like MAX(visited_at) lose the column's declared
// type, so the driver returns them as strings that we parse ourselves.
var sqliteTimeFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

// parseSQLiteTime parses a timestamp string produced by an aggregate query.
// It returns the zero time when no known layout matches.
func parseSQLiteTime(value string) time.Time {
	for _, layout := range sqliteTimeFormats {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}

// nullTime converts a nullable aggregate timestamp into a parsed time. It
// returns the zero time when the value is NULL.
func nullTime(value sql.NullString) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return parseSQLiteTime(value.String)
}
