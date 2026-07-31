package todo

import (
	"fmt"
	"strings"
	"time"

	"browser-server/internal/models"
)

// CreateInput is the shared contract for creating a Todo from HTTP or AI tool input.
type CreateInput struct {
	UserID      int            `json:"user_id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Domain      string         `json:"domain"`
	CaptureID   string         `json:"capture_id"`
	Rrule       string         `json:"rrule"`
	ParentID    int            `json:"parent_id"`
	Priority    string         `json:"priority"`
	Status      string         `json:"status"`
	Color       string         `json:"color"`
	Tags        []string       `json:"tags"`
	Subtasks    []SubtaskInput `json:"subtasks"`
	StartDate   *string        `json:"start_date"`
	EndDate     *string        `json:"end_date"`
}

// SubtaskInput represents a single subtask payload.
type SubtaskInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Color       string   `json:"color"`
	Tags        []string `json:"tags"`
}

// ValidationError contains field-level validation messages.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(e.Fields))
	for field, msg := range e.Fields {
		parts = append(parts, fmt.Sprintf("%s %s", field, msg))
	}
	return strings.Join(parts, "; ")
}

// UpdateInput is the shared contract for updating a Todo from HTTP or AI tool input.
type UpdateInput struct {
	UserID         int       `json:"user_id"`
	ID             int       `json:"id"`
	Title          *string   `json:"title"`
	Description    *string   `json:"description"`
	Status         *string   `json:"status"`
	Priority       *string   `json:"priority"`
	Color          *string   `json:"color"`
	Tags           *[]string `json:"tags"`
	Position       *int      `json:"position"`
	ParentID       *int      `json:"parent_id"`
	StartDate      *string   `json:"start_date"`
	EndDate        *string   `json:"end_date"`
	Rrule          *string   `json:"rrule"`
	Domain         *string   `json:"domain"`
	ScreenshotPath *string   `json:"screenshot_path"`
	Pinned         *bool     `json:"pinned"`
}

// CreateResult is the shared response payload for a created todo.
type CreateResult struct {
	ID          int64           `json:"id"`
	UserID      int             `json:"user_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	Priority    string          `json:"priority"`
	Color       *string         `json:"color"`
	Tags        []string        `json:"tags"`
	ParentID    *int            `json:"parent_id"`
	Position    int             `json:"position"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	StartDate   *string         `json:"start_date"`
	EndDate     *string         `json:"end_date"`
	Subtasks    []SubtaskResult `json:"subtasks,omitempty"`
	IsDuplicate bool            `json:"-"`
}

// SubtaskResult is the shared response payload for a created subtask.
type SubtaskResult struct {
	ID          int64     `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Color       *string   `json:"color"`
	Tags        []string  `json:"tags"`
	ParentID    int64     `json:"parent_id"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateResult is the shared response payload for an updated todo.
type UpdateResult struct {
	ID             int64     `json:"id"`
	UserID         int       `json:"user_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Domain         *string   `json:"domain"`
	ScreenshotPath *string   `json:"screenshot_path"`
	Pinned         bool      `json:"pinned"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	Color          *string   `json:"color"`
	Rrule          *string   `json:"rrule"`
	Tags           []string  `json:"tags"`
	ParentID       *int      `json:"parent_id"`
	Position       int       `json:"position"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	StartDate      *string   `json:"start_date"`
	EndDate        *string   `json:"end_date"`
}

// ListFilter specifies filtering and sorting for List queries.
type ListFilter struct {
	UserID       int
	Status       string
	Domain       string
	Priority     string
	Tag          string
	ParentID     int // 0 → top-level (parent_id IS NULL); >0 → specific parent
	SortField    string
	SortOrder    string
	ShowArchived bool
}

// Repository defines the data-access contract for todos.
type Repository interface {
	Create(input *CreateInput) (*CreateResult, error)
	Update(input *UpdateInput) (*UpdateResult, error)
	List(filter ListFilter) ([]models.TodoResponse, error)
	GetByID(id int) (*models.TodoResponse, error)
	Delete(id int) error
}
