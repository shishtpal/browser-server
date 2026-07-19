package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/ai/provider"
	"browser-server/internal/db"
)

const maxOutput = 32 * 1024

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Execute     func(context.Context, json.RawMessage) (any, error)
}

type Registry struct{ tools map[string]Tool }

func New() *Registry {
	r := &Registry{tools: map[string]Tool{}}
	r.add(Tool{Name: "get_current_time", Description: "Get the current server time", Schema: json.RawMessage(`{"type":"object","properties":{"timezone":{"type":"string"}},"additionalProperties":false}`), Execute: currentTime})
	r.add(Tool{Name: "search_bookmarks", Description: "Search the local bookmark database", Schema: json.RawMessage(`{"type":"object","properties":{"user_id":{"type":"integer","minimum":1},"query":{"type":"string","maxLength":200},"limit":{"type":"integer","minimum":1,"maximum":20}},"required":["user_id","query"],"additionalProperties":false}`), Execute: searchBookmarks})
	return r
}
func (r *Registry) add(t Tool) { r.tools[t.Name] = t }
func (r *Registry) Specs(allowed []string) []provider.ToolSpec {
	var out []provider.ToolSpec
	for _, n := range allowed {
		if t, ok := r.tools[n]; ok {
			out = append(out, provider.ToolSpec{Name: t.Name, Description: t.Description, Parameters: t.Schema})
		}
	}
	return out
}
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) ([]byte, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool")
	}
	v, err := t.Execute(ctx, args)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	if len(b) > maxOutput {
		return nil, fmt.Errorf("tool output exceeds limit")
	}
	return b, err
}

func currentTime(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Timezone string `json:"timezone"`
	}
	if err := strict(raw, &a, map[string]bool{"timezone": true}); err != nil {
		return nil, err
	}
	loc := time.UTC
	if a.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(a.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone")
		}
	}
	return map[string]string{"time": time.Now().In(loc).Format(time.RFC3339), "timezone": loc.String()}, nil
}
func searchBookmarks(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "limit": true}); err != nil {
		return nil, err
	}
	a.Query = strings.TrimSpace(a.Query)
	if a.UserID < 1 || a.Query == "" || len(a.Query) > 200 {
		return nil, fmt.Errorf("invalid bookmark search arguments")
	}
	if a.Limit == 0 {
		a.Limit = 10
	}
	if a.Limit < 1 || a.Limit > 20 {
		return nil, fmt.Errorf("limit must be 1 to 20")
	}
	rows, err := db.BookmarkDB.QueryContext(ctx, `SELECT id,title,url FROM bookmarks WHERE user_id=? AND (title LIKE ? OR url LIKE ? OR description LIKE ?) ORDER BY updated_at DESC LIMIT ?`, a.UserID, "%"+a.Query+"%", "%"+a.Query+"%", "%"+a.Query+"%", a.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int
		var title, url string
		if err := rows.Scan(&id, &title, &url); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "url": url})
	}
	return out, rows.Err()
}
func strict(raw json.RawMessage, dst any, allowed map[string]bool) error {
	var fields map[string]json.RawMessage
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("arguments must be a JSON object")
	}
	for k := range fields {
		if !allowed[k] {
			return fmt.Errorf("unknown argument %q", k)
		}
	}
	return json.Unmarshal(raw, dst)
}
