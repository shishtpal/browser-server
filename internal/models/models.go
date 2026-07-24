package models

import "time"

type Todo struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Domain         string     `json:"domain"`
	CaptureID      string     `json:"capture_id,omitempty"`
	ScreenshotPath string     `json:"screenshot_path"`
	Completed      bool       `json:"completed"`
	Priority       string     `json:"priority"`
	DueDate        *time.Time `json:"due_date"`
	Tags           string     `json:"tags"`
	ParentID       *int       `json:"parent_id"`
	Position       int        `json:"position"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type TodoResponse struct {
	Todo
	Tags     []string       `json:"tags"`
	Subtasks []TodoResponse `json:"subtasks,omitempty"`
}

type Screenshot struct {
	ID        int       `json:"id"`
	TodoID    int       `json:"todo_id"`
	Filename  string    `json:"filename"`
	CreatedAt time.Time `json:"created_at"`
}

type Bookmark struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Tags        string    `json:"tags"`
	FolderPath  string    `json:"folder_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BookmarkResponse struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	FolderPath  string    `json:"folder_path"`
	CaptureID   string    `json:"capture_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ImportResult struct {
	Imported  int                `json:"imported"`
	Skipped   int                `json:"skipped"`
	Bookmarks []BookmarkResponse `json:"bookmarks"`
}

type History struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	VisitedAt time.Time `json:"visited_at"`
	Duration  int       `json:"duration"`
}

type HistoryImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// GroupedHistoryEntry is a single URL aggregated across all of its visits.
type GroupedHistoryEntry struct {
	URL           string    `json:"url"`
	Title         string    `json:"title"`
	Count         int       `json:"count"`
	TotalDuration int       `json:"total_duration"`
	LastVisited   time.Time `json:"last_visited"`
}

// GroupedHistoryResponse is a server-paginated page of grouped history entries.
type GroupedHistoryResponse struct {
	Entries []GroupedHistoryEntry `json:"entries"`
	Total   int                   `json:"total"`
	Limit   int                   `json:"limit"`
	Offset  int                   `json:"offset"`
}

// HistoryDomainSummary aggregates all history visits for one hostname.
type HistoryDomainSummary struct {
	Domain        string    `json:"domain"`
	VisitCount    int       `json:"visit_count"`
	URLCount      int       `json:"url_count"`
	TotalDuration int       `json:"total_duration"`
	LastVisited   time.Time `json:"last_visited"`
}

// OmniboxSearchResult is a normalized bookmark/history suggestion for the
// browser extension omnibox integration.
type OmniboxSearchResult struct {
	Source      string     `json:"source"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Description string     `json:"description,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	FolderPath  string     `json:"folder_path,omitempty"`
	VisitCount  int        `json:"visit_count,omitempty"`
	LastVisited *time.Time `json:"last_visited,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type WalletEntry struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	Website       string    `json:"website"`
	LoginProvider string    `json:"login_provider"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WalletImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Route struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type DomainUsageEntry struct {
	Domain  string `json:"domain"`
	Date    string `json:"date"`
	Seconds int    `json:"seconds"`
}

type UsageBatchRequest struct {
	UserID  int                `json:"user_id"`
	Entries []DomainUsageEntry `json:"entries"`
}

type UsageBatchResponse struct {
	Upserted int `json:"upserted"`
}

type DomainUsage struct {
	Domain       string  `json:"domain"`
	TotalSeconds int     `json:"total_seconds"`
	Percentage   float64 `json:"percentage"`
}

type TimelinePoint struct {
	Period       string `json:"period"`
	TotalSeconds int    `json:"total_seconds"`
}

type AnalyticsSummary struct {
	TotalSeconds int             `json:"total_seconds"`
	Domains      []DomainUsage   `json:"domains"`
	Timeline     []TimelinePoint `json:"timeline"`
}
