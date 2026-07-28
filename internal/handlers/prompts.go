package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

// PromptFolder represents a prompt folder.
type PromptFolder = models.PromptFolder

// Prompt represents a prompt.
type Prompt = models.Prompt

// PromptResponse represents a prompt response with parsed tags.
type PromptResponse = models.PromptResponse

// ─── Prompt Folders ─────────────────────────────────────

func GetPromptFolders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	rows, err := db.PromptDB.Query("SELECT id, user_id, name, created_at, updated_at FROM prompt_folders WHERE user_id = ? ORDER BY name ASC", userID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	folders := []models.PromptFolder{}
	for rows.Next() {
		var folder models.PromptFolder
		err := rows.Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt)
		if err != nil {
			continue
		}
		folders = append(folders, folder)
	}

	json.NewEncoder(w).Encode(folders)
}

func CreatePromptFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input struct {
		UserID int    `json:"user_id"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("name", input.Name)
	v.MaxLength("name", input.Name, 100)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	now := time.Now()
	result, err := db.PromptDB.Exec("INSERT INTO prompt_folders (user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?)",
		input.UserID, input.Name, now, now)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	id, _ := result.LastInsertId()
	folder := models.PromptFolder{
		ID:        int(id),
		UserID:    input.UserID,
		Name:      input.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func UpdatePromptFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid folder id")
		return
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.Required("name", input.Name)
	v.MaxLength("name", input.Name, 100)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	var folder models.PromptFolder
	err := db.PromptDB.QueryRow("SELECT id, user_id, name, created_at, updated_at FROM prompt_folders WHERE id = ?", id).
		Scan(&folder.ID, &folder.UserID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Folder not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	now := time.Now()
	_, err = db.PromptDB.Exec("UPDATE prompt_folders SET name = ?, updated_at = ? WHERE id = ?", input.Name, now, id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	folder.Name = input.Name
	folder.UpdatedAt = now
	json.NewEncoder(w).Encode(folder)
}

func DeletePromptFolder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid folder id")
		return
	}

	_, err := db.PromptDB.Exec("UPDATE prompts SET folder_id = NULL WHERE folder_id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	result, err := db.PromptDB.Exec("DELETE FROM prompt_folders WHERE id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		helpers.WriteError(w, http.StatusNotFound, "Folder not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Prompts ────────────────────────────────────────────

func GetPrompts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	folderIDStr := r.URL.Query().Get("folder_id")
	query := r.URL.Query().Get("q")

	sqlQuery := `
		SELECT p.id, p.user_id, p.folder_id, p.title, p.content, p.description, p.tags,
			pf.name as folder_name, p.created_at, p.updated_at
		FROM prompts p
		LEFT JOIN prompt_folders pf ON p.folder_id = pf.id
		WHERE p.user_id = ?`
	args := []interface{}{userID}

	if folderIDStr != "" {
		folderID, _ := strconv.Atoi(folderIDStr)
		if folderID > 0 {
			sqlQuery += " AND p.folder_id = ?"
			args = append(args, folderID)
		} else {
			sqlQuery += " AND p.folder_id IS NULL"
		}
	}

	if query != "" {
		sqlQuery += " AND (p.title LIKE ? OR p.content LIKE ?)"
		like := "%" + query + "%"
		args = append(args, like, like)
	}

	sqlQuery += " ORDER BY p.created_at DESC"

	rows, err := db.PromptDB.Query(sqlQuery, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	prompts := []models.PromptResponse{}
	for rows.Next() {
		var p models.Prompt
		var folderName sql.NullString
		err := rows.Scan(
			&p.ID, &p.UserID, &p.FolderID, &p.Title, &p.Content, &p.Description, &p.Tags,
			&folderName, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}

		resp := models.PromptResponse{
			Prompt: p,
			Tags:   helpers.ParseTagsFromJSON(p.Tags),
		}
		if folderName.Valid {
			name := folderName.String
			resp.FolderName = &name
		}
		prompts = append(prompts, resp)
	}

	json.NewEncoder(w).Encode(prompts)
}

func CreatePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input struct {
		UserID      int      `json:"user_id"`
		FolderID    *int     `json:"folder_id"`
		Title       string   `json:"title"`
		Content     string   `json:"content"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", input.UserID)
	v.Required("title", input.Title)
	v.Required("content", input.Content)
	v.MaxLength("title", input.Title, 200)
	v.MaxLength("content", input.Content, 10000)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	if input.FolderID != nil && *input.FolderID > 0 {
		var folderUserID int
		err := db.PromptDB.QueryRow("SELECT user_id FROM prompt_folders WHERE id = ?", *input.FolderID).Scan(&folderUserID)
		if err == sql.ErrNoRows {
			helpers.WriteError(w, http.StatusBadRequest, "Folder not found")
			return
		} else if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if folderUserID != input.UserID {
			helpers.WriteError(w, http.StatusForbidden, "Folder does not belong to user")
			return
		}
	}

	tagsJSON := helpers.TagsToJSON(input.Tags)
	now := time.Now()
	result, err := db.PromptDB.Exec(
		"INSERT INTO prompts (user_id, folder_id, title, content, description, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		input.UserID, input.FolderID, input.Title, input.Content, input.Description, tagsJSON, now, now,
	)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	id, _ := result.LastInsertId()
	prompt := models.Prompt{
		ID:          int(id),
		UserID:      input.UserID,
		FolderID:    input.FolderID,
		Title:       input.Title,
		Content:     input.Content,
		Description: input.Description,
		Tags:        tagsJSON,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	resp := models.PromptResponse{
		Prompt: prompt,
		Tags:   input.Tags,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func GetPromptByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	var p models.Prompt
	var folderName sql.NullString
	err := db.PromptDB.QueryRow(`
		SELECT p.id, p.user_id, p.folder_id, p.title, p.content, p.description, p.tags,
			pf.name as folder_name, p.created_at, p.updated_at
		FROM prompts p
		LEFT JOIN prompt_folders pf ON p.folder_id = pf.id
		WHERE p.id = ?`, id).
		Scan(&p.ID, &p.UserID, &p.FolderID, &p.Title, &p.Content, &p.Description, &p.Tags,
			&folderName, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	resp := models.PromptResponse{
		Prompt: p,
		Tags:   helpers.ParseTagsFromJSON(p.Tags),
	}
	if folderName.Valid {
		name := folderName.String
		resp.FolderName = &name
	}

	json.NewEncoder(w).Encode(resp)
}

func UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	var p models.Prompt
	var tagsJSON string
	err := db.PromptDB.QueryRow("SELECT id, user_id, folder_id, title, content, description, tags, created_at, updated_at FROM prompts WHERE id = ?", id).
		Scan(&p.ID, &p.UserID, &p.FolderID, &p.Title, &p.Content, &p.Description, &p.Tags, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	var input struct {
		Title       *string  `json:"title"`
		Content     *string  `json:"content"`
		Description *string  `json:"description"`
		FolderID    *int     `json:"folder_id"`
		Tags        []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Title != nil {
		v := helpers.NewValidator()
		v.Required("title", *input.Title)
		v.MaxLength("title", *input.Title, 200)
		if !v.OK() {
			helpers.WriteValidationError(w, v.Fields())
			return
		}
		p.Title = *input.Title
	}
	if input.Content != nil {
		if len(*input.Content) > 10000 {
			helpers.WriteValidationError(w, map[string]string{"content": "must be 10000 characters or fewer"})
			return
		}
		p.Content = *input.Content
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.FolderID != nil {
		if *input.FolderID > 0 {
			var folderUserID int
			err := db.PromptDB.QueryRow("SELECT user_id FROM prompt_folders WHERE id = ?", *input.FolderID).Scan(&folderUserID)
			if err == sql.ErrNoRows {
				helpers.WriteError(w, http.StatusBadRequest, "Folder not found")
				return
			} else if err != nil {
				helpers.WriteError(w, http.StatusInternalServerError, "Database error")
				return
			}
			if folderUserID != p.UserID {
				helpers.WriteError(w, http.StatusForbidden, "Folder does not belong to user")
				return
			}
			p.FolderID = input.FolderID
		} else {
			p.FolderID = nil
		}
	}
	if input.Tags != nil {
		tagsJSON = helpers.TagsToJSON(input.Tags)
	}

	_, err = db.PromptDB.Exec(
		"UPDATE prompts SET title = ?, content = ?, description = ?, folder_id = ?, tags = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		p.Title, p.Content, p.Description, p.FolderID, tagsJSON, id,
	)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	p.Tags = tagsJSON
	resp := models.PromptResponse{
		Prompt: p,
		Tags:   helpers.ParseTagsFromJSON(tagsJSON),
	}
	json.NewEncoder(w).Encode(resp)
}

func DeletePrompt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := helpers.GetIDFromPath(r)
	if id == 0 {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid prompt id")
		return
	}

	result, err := db.PromptDB.Exec("DELETE FROM prompts WHERE id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		helpers.WriteError(w, http.StatusNotFound, "Prompt not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ─── Fuzzy Search ───────────────────────────────────────

func SearchPrompts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	if userID <= 0 {
		helpers.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := helpers.GetLimitFromQuery(r, 10)
	if limit > 20 {
		limit = 20
	}

	args := []interface{}{userID}
	var whereClauses []string

	if query != "" {
		whereClauses = append(whereClauses, "(p.title LIKE ? OR p.content LIKE ?)")
		like := "%" + query + "%"
		args = append(args, like, like)
	}

	whereSQL := strings.Join(whereClauses, " AND ")
	if whereSQL != "" {
		whereSQL = " AND " + whereSQL
	}

	sqlQuery := `
		SELECT p.id, p.user_id, p.folder_id, p.title, p.content, p.description, p.tags,
			pf.name as folder_name, p.created_at, p.updated_at
		FROM prompts p
		LEFT JOIN prompt_folders pf ON p.folder_id = pf.id
		WHERE p.user_id = ?` + whereSQL + `
		ORDER BY
			CASE WHEN p.title LIKE ? THEN 0 ELSE 1 END,
			p.created_at DESC
		LIMIT ?`

	titleLike := "%" + query + "%"
	args = append(args, titleLike, limit)

	rows, err := db.PromptDB.Query(sqlQuery, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	prompts := []models.PromptResponse{}
	for rows.Next() {
		var p models.Prompt
		var folderName sql.NullString
		err := rows.Scan(
			&p.ID, &p.UserID, &p.FolderID, &p.Title, &p.Content, &p.Description, &p.Tags,
			&folderName, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			continue
		}

		resp := models.PromptResponse{
			Prompt: p,
			Tags:   helpers.ParseTagsFromJSON(p.Tags),
		}
		if folderName.Valid {
			name := folderName.String
			resp.FolderName = &name
		}
		prompts = append(prompts, resp)
	}

	json.NewEncoder(w).Encode(prompts)
}
