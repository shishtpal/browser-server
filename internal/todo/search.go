package todo

import (
	"context"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
)

// SearchFilter expresses structured filters for the shared search engine.
type SearchFilter struct {
	UserID      int
	Status      string
	Priority    string
	Tags        []string
	StartDate   string
	EndDate     string
	Scheduled   bool
	OrderByDate bool
}

// TodoCandidate is the search-engine candidate type for todos and calendar.
type TodoCandidate = searchengine.Candidate[models.Todo]

// TodoCandidateSet is a set of todo candidates.
type TodoCandidateSet = searchengine.CandidateSet[models.Todo]

// SearchCandidates loads todo candidates filtered by the caller, returning a
// candidate for every row that passes the structured filter. It is owned by the
// todo domain package and uses TodoColumns / ScanTodo.
func SearchCandidates(ctx context.Context, filter SearchFilter, req searchengine.CandidateRequest) (TodoCandidateSet, error) {
	where := []string{"user_id = ?"}
	args := []any{filter.UserID}

	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		where = append(where, "priority = ?")
		args = append(args, filter.Priority)
	}
	if filter.Scheduled {
		where = append(where, "start_date IS NOT NULL")
	}
	if filter.StartDate != "" {
		where = append(where, "start_date >= ?")
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		where = append(where, "(end_date <= ? OR (end_date IS NULL AND start_date <= ?))")
		args = append(args, filter.EndDate, filter.EndDate)
	}
	for _, tag := range filter.Tags {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(todos.tags) WHERE json_each.value = ?)")
		args = append(args, tag)
	}

	order := "updated_at DESC, id DESC"
	if filter.OrderByDate {
		order = "start_date ASC, id ASC"
	}

	limit := req.MaxCandidates
	if limit < 1 {
		limit = searchengine.DefaultCandidateCap
	}
	args = append(args, limit+1)

	q := fmt.Sprintf("SELECT %s FROM todos WHERE %s ORDER BY %s LIMIT ?", TodoColumns, strings.Join(where, " AND "), order)

	rows, err := db.TodoDB.QueryContext(ctx, q, args...)
	if err != nil {
		return TodoCandidateSet{}, err
	}
	defer rows.Close()

	var candidates []TodoCandidate
	i := 0
	for rows.Next() {
		todoRow, tagsJSON, err := ScanTodo(rows)
		if err != nil {
			return TodoCandidateSet{}, err
		}
		tags := helpers.ParseTagsFromJSON(tagsJSON)
		if tags == nil {
			tags = []string{}
		}
		tagsText := strings.Join(tags, " ")
		sd := ""
		if todoRow.StartDate != nil {
			sd = todoRow.StartDate.Format("2006-01-02")
		}
		candidates = append(candidates, TodoCandidate{
			Key: fmt.Sprintf("todo:%d", todoRow.ID),
			Fields: []searchengine.Field{
				{Name: "title", Text: todoRow.Title, Weight: 10},
				{Name: "description", Text: todoRow.Description, Weight: 3},
				{Name: "tags", Text: tagsText, Weight: 5},
				{Name: "start_date", Text: sd, Weight: 1},
			},
			Value:      todoRow,
			SourceRank: i,
		})
		i++
	}
	if err := rows.Err(); err != nil {
		return TodoCandidateSet{}, err
	}
	truncated := len(candidates) > limit
	if truncated {
		candidates = candidates[:limit]
	}
	return TodoCandidateSet{Candidates: candidates, Truncated: truncated}, nil
}

// TodoSearchHitMap renders a scored Todo hit as the map the search_todos tool
// returns. Optional fields keep their existing omission/null behavior.
func TodoSearchHitMap(t models.Todo, tags []string, score float64) map[string]any {
	entry := map[string]any{
		"id":          t.ID,
		"title":       t.Title,
		"description": t.Description,
		"status":      t.Status,
		"priority":    t.Priority,
		"pinned":      t.Pinned,
		"tags":        tags,
		"score":       score,
	}
	if t.StartDate != nil {
		entry["start_date"] = t.StartDate.Format("2006-01-02")
	}
	if t.EndDate != nil {
		entry["end_date"] = t.EndDate.Format("2006-01-02")
	}
	return entry
}

// CalendarSearchHitMap renders a scored Todo hit as the map the search_calendar
// tool returns.
func CalendarSearchHitMap(t models.Todo, tags []string, score float64) map[string]any {
	entry := TodoSearchHitMap(t, tags, score)
	if t.Rrule != "" {
		entry["rrule"] = t.Rrule
	}
	return entry
}
