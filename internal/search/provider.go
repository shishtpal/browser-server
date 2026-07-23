package search

import (
	"context"
	"time"
)

type Provider interface {
	Name() string
	Search(context.Context, SearchQuery) (*SearchResponse, error)
	IsAvailable(context.Context) bool
	RequiresAPIKey() bool
}

type TimeRange string

const (
	TimeAny   TimeRange = ""
	TimeDay   TimeRange = "day"
	TimeWeek  TimeRange = "week"
	TimeMonth TimeRange = "month"
	TimeYear  TimeRange = "year"
)

type SearchQuery struct {
	Query                          string
	Count                          int
	Country, Language              string
	TimeRange                      TimeRange
	SafeSearch                     bool
	IncludeDomains, ExcludeDomains []string
}

func (q *SearchQuery) GetDefaults() *SearchQuery {
	if q.Count <= 0 {
		q.Count = 10
	}
	if q.Count > 20 {
		q.Count = 20
	}
	if q.Country == "" {
		q.Country = "us"
	}
	if q.Language == "" {
		q.Language = "en"
	}
	return q
}

type SearchResult struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Snippet     string     `json:"snippet"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Source      string     `json:"source"`
	Score       float64    `json:"score,omitempty"`
}
type SearchResponse struct {
	Results      []SearchResult `json:"results"`
	Query        string         `json:"query"`
	Provider     string         `json:"provider"`
	Duration     time.Duration  `json:"duration"`
	TotalResults int            `json:"total_results,omitempty"`
}
type FetchOptions struct {
	MaxBytes        int64
	Timeout         int
	FollowRedirects bool
	ExtractContent  bool
	UserAgent       string
}
type FetchResult struct {
	URL         string        `json:"url"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	RawHTML     string        `json:"raw_html,omitempty"`
	TextContent string        `json:"text_content"`
	StatusCode  int           `json:"status_code"`
	ContentType string        `json:"content_type"`
	Size        int64         `json:"size"`
	FetchTime   time.Duration `json:"fetch_time"`
	Error       string        `json:"error,omitempty"`
}
