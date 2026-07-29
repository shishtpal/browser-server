package bookmark

import (
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func Response(b models.Bookmark) models.BookmarkResponse {
	return models.BookmarkResponse{
		ID:          b.ID,
		UserID:      b.UserID,
		Title:       b.Title,
		URL:         b.URL,
		Description: b.Description,
		Tags:        helpers.ParseTagsFromJSON(b.Tags),
		FolderPath:  b.FolderPath,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func Responses(bookmarks []models.Bookmark) []models.BookmarkResponse {
	out := make([]models.BookmarkResponse, 0, len(bookmarks))
	for _, b := range bookmarks {
		out = append(out, Response(b))
	}
	return out
}

func SearchMap(b models.Bookmark) map[string]any {
	return map[string]any{"id": b.ID, "title": b.Title, "url": b.URL}
}
