package tools

import (
	"context"
	_ "embed"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"browser-server/internal/db"
)

//go:embed schemas/search_prompts.json
var searchPromptsSchema []byte

func registerSearchPrompts(r *Registry) {
	r.add(Tool{
		Name:        "search_prompts",
		Category:    "General",
		Description: "Search the local prompt database. Can filter by user and text query across title/content.",
		Schema:      json.RawMessage(searchPromptsSchema),
		Execute:     searchPrompts,
	})
}

func searchPrompts(ctx context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		UserID int    `json:"user_id"`
		Query  string `json:"query"`
		Limit  int    `json:"limit"`
	}
	if err := strict(raw, &a, map[string]bool{"user_id": true, "query": true, "limit": true}); err != nil {
		return nil, err
	}
	if a.UserID < 1 {
		return nil, fmt.Errorf("user_id is required")
	}
	a.Query = strings.TrimSpace(a.Query)
	if len(a.Query) > 200 {
		return nil, fmt.Errorf("query too long")
	}
	if a.Limit == 0 {
		a.Limit = 5
	}
	if a.Limit < 1 || a.Limit > 10 {
		return nil, fmt.Errorf("limit must be 1 to 10")
	}

	where := []string{"p.user_id = ?"}
	args := []any{a.UserID}

	if a.Query != "" {
		where = append(where, "(p.title LIKE ? OR p.content LIKE ?)")
		args = append(args, "%"+a.Query+"%", "%"+a.Query+"%")
	}

	titleLike := "%" + a.Query + "%"
	args = append(args, titleLike, a.Limit)

	q := fmt.Sprintf(
		`SELECT p.id, p.title, p.content, p.description, pf.name as folder_name
		 FROM prompts p
		 LEFT JOIN prompt_folders pf ON p.folder_id = pf.id
		 WHERE %s
		 ORDER BY CASE WHEN p.title LIKE ? THEN 0 ELSE 1 END, p.created_at DESC
		 LIMIT ?`,
		strings.Join(where, " AND "),
	)

	rows, err := db.PromptDB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var id int
		var title, content, description string
		var folderName sql.NullString
		if err := rows.Scan(&id, &title, &content, &description, &folderName); err != nil {
			return nil, err
		}
		entry := map[string]any{
			"id":          id,
			"title":       title,
			"content":     content,
			"description": description,
		}
		if folderName.Valid {
			entry["folder_name"] = folderName.String
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}
