package models

import "time"

type Todo struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Domain         string    `json:"domain"`
	ScreenshotPath string    `json:"screenshot_path"`
	Completed      bool      `json:"completed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

type WalletEntry struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
