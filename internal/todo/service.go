package todo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

// TodoColumns is the column list used in all SELECT queries for todos.
const TodoColumns = "id, user_id, title, description, domain, screenshot_path, pinned, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at"

// NullIfEmpty returns a pointer to the string if non-blank, or nil.
func NullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// nullIfEmptyAny returns an any value suitable for SQL params: nil for empty, or the string.
func nullIfEmptyAny(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// nullTimeToAny returns nil or the time for SQL params.
func nullTimeToAny(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// nullAnyFromPtr returns nil or the pointed value (for *string).
func nullAnyFromPtr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// nullAnyFromIntPtr returns nil or the pointed value (for *int).
func nullAnyFromIntPtr(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// SQLRepository is the real implementation backed by *sql.DB.
type SQLRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) Create(input *CreateInput) (*CreateResult, error) {
	if err := ValidateCreateInput(input); err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	var startDate *time.Time
	if input.StartDate != nil && *input.StartDate != "" {
		startDate = ParseDate(*input.StartDate)
		if startDate == nil {
			return nil, &ValidationError{Fields: map[string]string{"start_date": "must be a valid date"}}
		}
	}
	var endDate *time.Time
	if input.EndDate != nil && *input.EndDate != "" {
		endDate = ParseDate(*input.EndDate)
		if endDate == nil {
			return nil, &ValidationError{Fields: map[string]string{"end_date": "must be a valid date"}}
		}
	}

	var parentID *int
	if input.ParentID > 0 {
		var parentUserID int
		err := r.db.QueryRow("SELECT user_id FROM todos WHERE id = ?", input.ParentID).Scan(&parentUserID)
		if err == sql.ErrNoRows {
			return nil, &ValidationError{Fields: map[string]string{"parent_id": fmt.Sprintf("todo %d not found", input.ParentID)}}
		}
		if err != nil {
			return nil, err
		}
		if parentUserID != input.UserID {
			return nil, &ValidationError{Fields: map[string]string{"parent_id": fmt.Sprintf("todo %d does not belong to user %d", input.ParentID, input.UserID)}}
		}
		parentID = &input.ParentID
	}

	tagsJSON := helpers.TagsToJSON(input.Tags)
	var whereClause string
	var posArgs []any
	if parentID != nil {
		whereClause = "WHERE parent_id = ? AND user_id = ?"
		posArgs = []any{*parentID, input.UserID}
	} else {
		whereClause = "WHERE parent_id IS NULL AND user_id = ?"
		posArgs = []any{input.UserID}
	}
	var maxPos sql.NullInt64
	if err := r.db.QueryRow("SELECT COALESCE(MAX(position), -1) FROM todos "+whereClause, posArgs...).Scan(&maxPos); err != nil {
		return nil, err
	}
	position := int(maxPos.Int64) + 1

	captureID := NullIfEmpty(input.CaptureID)
	rrule := NullIfEmpty(input.Rrule)
	result, err := r.db.Exec(`
INSERT INTO todos (user_id, title, description, domain, capture_id, status, priority, color, start_date, end_date, rrule, tags, parent_id, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(user_id, capture_id) DO NOTHING`,
		input.UserID, input.Title, input.Description, nullIfEmptyAny(input.Domain),
		nullAnyFromPtr(captureID), input.Status, input.Priority, strings.TrimSpace(input.Color),
		nullTimeToAny(startDate), nullTimeToAny(endDate), nullAnyFromPtr(rrule),
		tagsJSON, nullAnyFromIntPtr(parentID), position)
	if err != nil {
		return nil, err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 && captureID != nil {
		// Duplicate capture — return existing todo.
		row := r.db.QueryRow("SELECT "+TodoColumns+" FROM todos WHERE user_id = ? AND capture_id = ?", input.UserID, *captureID)
		var todoRow models.Todo
		var tagsJSONStr string
		var sd sql.NullTime
		var ed sql.NullTime
		var pid sql.NullInt64
		var rruleStr sql.NullString
		var dom sql.NullString
		var scPath sql.NullString
		var col sql.NullString
		if err := row.Scan(
			&todoRow.ID, &todoRow.UserID, &todoRow.Title, &todoRow.Description, &dom,
			&scPath, &todoRow.Pinned, &todoRow.Status, &todoRow.Priority, &col,
			&sd, &ed, &rruleStr, &tagsJSONStr, &pid,
			&todoRow.Position, &todoRow.CreatedAt, &todoRow.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if dom.Valid {
			todoRow.Domain = dom.String
		}
		if scPath.Valid {
			todoRow.ScreenshotPath = scPath.String
		}
		if col.Valid {
			todoRow.Color = col.String
		}
		if sd.Valid {
			todoRow.StartDate = &sd.Time
		}
		if ed.Valid {
			todoRow.EndDate = &ed.Time
		}
		if pid.Valid {
			p := int(pid.Int64)
			todoRow.ParentID = &p
		}
		if rruleStr.Valid {
			todoRow.Rrule = rruleStr.String
		}
		tags := helpers.ParseTagsFromJSON(tagsJSONStr)
		if tags == nil {
			tags = []string{}
		}
		return &CreateResult{
			ID:          int64(todoRow.ID),
			UserID:      todoRow.UserID,
			Title:       todoRow.Title,
			Description: todoRow.Description,
			Status:      todoRow.Status,
			Priority:    todoRow.Priority,
			Color:       NullIfEmpty(todoRow.Color),
			Tags:        tags,
			ParentID:    todoRow.ParentID,
			Position:    todoRow.Position,
			CreatedAt:   todoRow.CreatedAt,
			UpdatedAt:   todoRow.UpdatedAt,
			IsDuplicate: true,
		}, nil
	}

	todoID, _ := result.LastInsertId()
	resultPayload := &CreateResult{
		ID:          todoID,
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		Status:      input.Status,
		Priority:    input.Priority,
		Color:       NullIfEmpty(input.Color),
		Tags:        input.Tags,
		ParentID:    parentID,
		Position:    position,
		CreatedAt:   now,
		UpdatedAt:   now,
		StartDate:   nil,
		EndDate:     nil,
	}
	if startDate != nil {
		s := startDate.Format("2006-01-02")
		resultPayload.StartDate = &s
	}
	if endDate != nil {
		e := endDate.Format("2006-01-02")
		resultPayload.EndDate = &e
	}
	if len(input.Subtasks) > 0 {
		resultPayload.Subtasks = make([]SubtaskResult, 0, len(input.Subtasks))
		for i, st := range input.Subtasks {
			stPriority := "medium"
			if st.Priority != "" {
				stPriority = st.Priority
			}
			stStatus := "pending"
			if st.Status != "" {
				stStatus = st.Status
			}
			stColor := NullIfEmpty(strings.TrimSpace(st.Color))
			stTags := st.Tags
			if stTags == nil {
				stTags = []string{}
			}
			stTagsJSON := helpers.TagsToJSON(stTags)
			stResult, err := r.db.Exec(`
INSERT INTO todos (user_id, title, description, status, priority, color, tags, parent_id, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
				input.UserID, st.Title, st.Description, stStatus, stPriority, nullAnyFromPtr(stColor), stTagsJSON, todoID, i)
			if err != nil {
				return nil, fmt.Errorf("failed to create subtask %d: %w", i, err)
			}
			stID, _ := stResult.LastInsertId()
			resultPayload.Subtasks = append(resultPayload.Subtasks, SubtaskResult{
				ID:          stID,
				UserID:      input.UserID,
				Title:       st.Title,
				Description: st.Description,
				Status:      stStatus,
				Priority:    stPriority,
				Color:       stColor,
				Tags:        stTags,
				ParentID:    todoID,
				Position:    i,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
	}
	return resultPayload, nil
}

func (r *SQLRepository) Update(input *UpdateInput) (*UpdateResult, error) {
	if err := ValidateUpdateInput(input); err != nil {
		return nil, err
	}

	var existingUserID int
	err := r.db.QueryRow("SELECT user_id FROM todos WHERE id = ?", input.ID).Scan(&existingUserID)
	if err == sql.ErrNoRows {
		return nil, &ValidationError{Fields: map[string]string{"id": fmt.Sprintf("todo %d not found", input.ID)}}
	}
	if err != nil {
		return nil, err
	}
	if existingUserID != input.UserID {
		return nil, &ValidationError{Fields: map[string]string{"user_id": fmt.Sprintf("todo %d does not belong to user %d", input.ID, input.UserID)}}
	}

	var startDate *time.Time
	if input.StartDate != nil {
		if *input.StartDate == "" {
			startDate = nil
		} else {
			startDate = ParseDate(*input.StartDate)
			if startDate == nil {
				return nil, &ValidationError{Fields: map[string]string{"start_date": "must be a valid date"}}
			}
		}
	}
	var endDate *time.Time
	if input.EndDate != nil {
		if *input.EndDate == "" {
			endDate = nil
		} else {
			endDate = ParseDate(*input.EndDate)
			if endDate == nil {
				return nil, &ValidationError{Fields: map[string]string{"end_date": "must be a valid date"}}
			}
		}
	}

	var parentID *int
	if input.ParentID != nil {
		if *input.ParentID == input.ID {
			return nil, &ValidationError{Fields: map[string]string{"parent_id": "must not equal the todo's own id"}}
		}
		if *input.ParentID == 0 {
			parentID = nil
		} else {
			var parentUserID int
			var grandparentID sql.NullInt64
			perr := r.db.QueryRow("SELECT user_id, parent_id FROM todos WHERE id = ?", *input.ParentID).Scan(&parentUserID, &grandparentID)
			if perr == sql.ErrNoRows {
				return nil, &ValidationError{Fields: map[string]string{"parent_id": fmt.Sprintf("todo %d not found", *input.ParentID)}}
			}
			if perr != nil {
				return nil, perr
			}
			if parentUserID != input.UserID {
				return nil, &ValidationError{Fields: map[string]string{"parent_id": fmt.Sprintf("todo %d does not belong to user %d", *input.ParentID, input.UserID)}}
			}
			if grandparentID.Valid {
				return nil, &ValidationError{Fields: map[string]string{"parent_id": fmt.Sprintf("todo %d is itself a subtask; nested subtasks are not allowed", *input.ParentID)}}
			}
			parentID = input.ParentID
		}
	}

	setClauses := []string{}
	args := []any{}
	if input.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *input.Title)
	}
	if input.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *input.Description)
	}
	if input.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, *input.Status)
	}
	if input.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *input.Priority)
	}
	if input.Color != nil {
		setClauses = append(setClauses, "color = ?")
		args = append(args, nullIfEmptyAny(*input.Color))
	}
	if input.Tags != nil {
		setClauses = append(setClauses, "tags = ?")
		args = append(args, helpers.TagsToJSON(*input.Tags))
	}
	if input.Position != nil {
		setClauses = append(setClauses, "position = ?")
		args = append(args, *input.Position)
	}
	if input.ParentID != nil {
		setClauses = append(setClauses, "parent_id = ?")
		args = append(args, nullAnyFromIntPtr(parentID))
	}
	if input.StartDate != nil {
		setClauses = append(setClauses, "start_date = ?")
		args = append(args, nullTimeToAny(startDate))
	}
	if input.EndDate != nil {
		setClauses = append(setClauses, "end_date = ?")
		args = append(args, nullTimeToAny(endDate))
	}
	if input.Rrule != nil {
		setClauses = append(setClauses, "rrule = ?")
		args = append(args, nullIfEmptyAny(*input.Rrule))
	}
	if input.Domain != nil {
		setClauses = append(setClauses, "domain = ?")
		args = append(args, nullIfEmptyAny(*input.Domain))
	}
	if input.ScreenshotPath != nil {
		setClauses = append(setClauses, "screenshot_path = ?")
		args = append(args, nullIfEmptyAny(*input.ScreenshotPath))
	}
	if input.Pinned != nil {
		setClauses = append(setClauses, "pinned = ?")
		args = append(args, *input.Pinned)
	}
	if len(setClauses) == 0 {
		return nil, &ValidationError{Fields: map[string]string{"_body": "no updatable fields provided"}}
	}
	args = append(args, input.ID)
	_, err = r.db.Exec("UPDATE todos SET "+strings.Join(setClauses, ", ")+", updated_at = CURRENT_TIMESTAMP WHERE id = ?", args...)
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRow("SELECT "+TodoColumns+" FROM todos WHERE id = ?", input.ID)
	todoRow, tagsJSON, err := ScanTodo(row)
	if err != nil {
		return nil, err
	}
	return todoToUpdateResult(todoRow, tagsJSON), nil
}

func (r *SQLRepository) List(filter ListFilter) ([]models.TodoResponse, error) {
	query := "SELECT " + TodoColumns + " FROM todos WHERE 1=1"
	args := []any{}

	if !filter.ShowArchived {
		query += " AND status != 'archived'"
	}

	if filter.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, filter.UserID)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.Domain != "" {
		query += " AND domain = ?"
		args = append(args, filter.Domain)
	}

	if filter.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filter.Priority)
	}

	if filter.Tag != "" {
		query += " AND tags LIKE ?"
		args = append(args, "%"+filter.Tag+"%")
	}

	if filter.ParentID == 0 {
		query += " AND parent_id IS NULL"
	} else {
		query += " AND parent_id = ?"
		args = append(args, filter.ParentID)
	}

	switch filter.SortField {
	case "position", "start_date", "created_at":
		query += " ORDER BY pinned DESC, " + filter.SortField
	case "priority":
		query += " ORDER BY pinned DESC, CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END"
	case "title":
		query += " ORDER BY pinned DESC, title"
	default:
		query += " ORDER BY pinned DESC, position"
	}
	if filter.SortOrder == "desc" {
		query += " DESC"
	} else {
		query += " ASC"
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []models.TodoResponse{}
	for rows.Next() {
		t, tagsJSON, err := ScanTodo(rows)
		if err != nil {
			continue
		}
		resp := TodoToResponse(t, tagsJSON)
		if filter.Tag == "" {
			childRows, cerr := r.db.Query("SELECT "+TodoColumns+" FROM todos WHERE parent_id = ? AND status != 'archived' ORDER BY pinned DESC, position ASC", t.ID)
			children := []models.TodoResponse{}
			if cerr == nil {
				for childRows.Next() {
					child, childTags, err := ScanTodo(childRows)
					if err == nil {
						children = append(children, TodoToResponse(child, childTags))
					}
				}
				childRows.Close()
			}
			resp.Subtasks = children
		}
		todos = append(todos, resp)
	}
	return todos, rows.Err()
}

func (r *SQLRepository) GetByID(id int) (*models.TodoResponse, error) {
	row := r.db.QueryRow("SELECT "+TodoColumns+" FROM todos WHERE id = ?", id)
	t, tagsJSON, err := ScanTodo(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	resp := TodoToResponse(t, tagsJSON)

	childRows, err := r.db.Query("SELECT "+TodoColumns+" FROM todos WHERE parent_id = ? ORDER BY pinned DESC, position ASC", t.ID)
	children := []models.TodoResponse{}
	if err == nil {
		for childRows.Next() {
			child, childTags, err := ScanTodo(childRows)
			if err == nil {
				children = append(children, TodoToResponse(child, childTags))
			}
		}
		childRows.Close()
	}
	resp.Subtasks = children
	return &resp, nil
}

func (r *SQLRepository) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM todos WHERE parent_id = ?", id); err != nil {
		return err
	}
	result, err := tx.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("not found")
	}
	return tx.Commit()
}

// ScanTodo scans a single row into a models.Todo and its tags JSON string.
func ScanTodo(scanner interface{ Scan(...any) error }) (models.Todo, string, error) {
	var todoRow models.Todo
	var tagsJSON string
	var startDate sql.NullTime
	var endDate sql.NullTime
	var parentID sql.NullInt64
	var domain sql.NullString
	var screenshotPath sql.NullString
	var color sql.NullString
	var rrule sql.NullString
	err := scanner.Scan(
		&todoRow.ID, &todoRow.UserID, &todoRow.Title, &todoRow.Description, &domain,
		&screenshotPath, &todoRow.Pinned, &todoRow.Status, &todoRow.Priority, &color,
		&startDate, &endDate, &rrule, &tagsJSON, &parentID,
		&todoRow.Position, &todoRow.CreatedAt, &todoRow.UpdatedAt,
	)
	if err != nil {
		return todoRow, tagsJSON, err
	}
	if domain.Valid {
		todoRow.Domain = domain.String
	}
	if screenshotPath.Valid {
		todoRow.ScreenshotPath = screenshotPath.String
	}
	if color.Valid {
		todoRow.Color = color.String
	}
	if rrule.Valid {
		todoRow.Rrule = rrule.String
	}
	if startDate.Valid {
		t := startDate.Time
		todoRow.StartDate = &t
	}
	if endDate.Valid {
		t := endDate.Time
		todoRow.EndDate = &t
	}
	if parentID.Valid {
		pid := int(parentID.Int64)
		todoRow.ParentID = &pid
	}
	return todoRow, tagsJSON, nil
}

// todoToUpdateResult converts a models.Todo and tags JSON into an UpdateResult.
func todoToUpdateResult(todoRow models.Todo, tagsJSON string) *UpdateResult {
	tags := helpers.ParseTagsFromJSON(tagsJSON)
	if tags == nil {
		tags = []string{}
	}
	var startDate *string
	if todoRow.StartDate != nil {
		s := todoRow.StartDate.Format("2006-01-02")
		startDate = &s
	}
	var endDate *string
	if todoRow.EndDate != nil {
		e := todoRow.EndDate.Format("2006-01-02")
		endDate = &e
	}
	return &UpdateResult{
		ID:             int64(todoRow.ID),
		UserID:         todoRow.UserID,
		Title:          todoRow.Title,
		Description:    todoRow.Description,
		Domain:         NullIfEmpty(todoRow.Domain),
		ScreenshotPath: NullIfEmpty(todoRow.ScreenshotPath),
		Pinned:         todoRow.Pinned,
		Status:         todoRow.Status,
		Priority:       todoRow.Priority,
		Color:          NullIfEmpty(todoRow.Color),
		Rrule:          NullIfEmpty(todoRow.Rrule),
		Tags:           tags,
		ParentID:       todoRow.ParentID,
		Position:       todoRow.Position,
		CreatedAt:      todoRow.CreatedAt,
		UpdatedAt:      todoRow.UpdatedAt,
		StartDate:      startDate,
		EndDate:        endDate,
	}
}

// TodoToResponse converts a models.Todo and tags JSON string to a models.TodoResponse.
func TodoToResponse(t models.Todo, tagsJSON string) models.TodoResponse {
	return models.TodoResponse{
		Todo:     t,
		Tags:     helpers.ParseTagsFromJSON(tagsJSON),
		Subtasks: []models.TodoResponse{},
	}
}
