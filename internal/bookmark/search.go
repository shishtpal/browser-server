package bookmark

import (
	"context"
	"fmt"
	"strings"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
	"browser-server/internal/searchengine"
)

// BookmarkCandidate is the search-engine candidate type for bookmarks.
type BookmarkCandidate = searchengine.Candidate[models.Bookmark]

// BookmarkCandidateSet is a set of bookmark candidates.
type BookmarkCandidateSet = searchengine.CandidateSet[models.Bookmark]

// SearchCandidates loads bookmarks for fuzzy ranking. It does not apply the
// final text LIKE or page limit, but it does apply ownership and tag/folder
// structured filters. It is intended for the search_bookmarks AI tool.
func SearchCandidates(ctx context.Context, userID int, tags []string, folderPathPrefix string, maxCandidates int) (BookmarkCandidateSet, error) {
	where := []string{"user_id = ?"}
	args := []any{userID}

	if folderPathPrefix != "" {
		where = append(where, "folder_path LIKE ?")
		args = append(args, folderPathPrefix+"%")
	}
	for _, tag := range tags {
		where = append(where, "EXISTS (SELECT 1 FROM json_each(bookmarks.tags) WHERE json_each.value = ?)")
		args = append(args, tag)
	}

	// updated_at has second precision, so ties are broken by id DESC to keep
	// the most recently inserted bookmarks first deterministically.
	query := "SELECT " + Columns + " FROM bookmarks WHERE " + strings.Join(where, " AND ") + " ORDER BY updated_at DESC, id DESC"
	if maxCandidates > 0 {
		query += " LIMIT ?"
		args = append(args, maxCandidates+1)
	}

	rows, err := db.BookmarkDB.QueryContext(ctx, query, args...)
	if err != nil {
		return BookmarkCandidateSet{}, err
	}
	bookmarks, err := ScanAll(rows)
	if err != nil {
		return BookmarkCandidateSet{}, err
	}

	truncated := maxCandidates > 0 && len(bookmarks) > maxCandidates
	if truncated {
		bookmarks = bookmarks[:maxCandidates]
	}

	candidates := make([]BookmarkCandidate, len(bookmarks))
	for i, b := range bookmarks {
		btags := helpers.ParseTagsFromJSON(b.Tags)
		if btags == nil {
			btags = []string{}
		}
		candidates[i] = BookmarkCandidate{
			Key: fmt.Sprintf("bookmark:%d", b.ID),
			Fields: []searchengine.Field{
				{Name: "title", Text: b.Title, Weight: 10},
				{Name: "url", Text: b.URL, Weight: 7},
				{Name: "description", Text: b.Description, Weight: 3},
				{Name: "tags", Text: strings.Join(btags, " "), Weight: 5},
				{Name: "folder_path", Text: b.FolderPath, Weight: 2},
			},
			Value:      b,
			SourceRank: i,
		}
	}
	return BookmarkCandidateSet{Candidates: candidates, Truncated: truncated}, nil
}

// SearchHitMap renders a scored bookmark as the map the search_bookmarks tool
// returns. It mirrors the full bookmark shape (bookmark.Response) plus a
// relevance score so descriptions and tags stay available to consumers.
func SearchHitMap(b models.Bookmark, score float64) map[string]any {
	tags := helpers.ParseTagsFromJSON(b.Tags)
	if tags == nil {
		tags = []string{}
	}
	return map[string]any{
		"id":          b.ID,
		"title":       b.Title,
		"url":         b.URL,
		"description": b.Description,
		"tags":        tags,
		"folder_path": b.FolderPath,
		"created_at":  b.CreatedAt,
		"updated_at":  b.UpdatedAt,
		"score":       score,
	}
}
