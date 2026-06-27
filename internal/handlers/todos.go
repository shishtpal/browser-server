package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func GetTodos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := helpers.GetUserIDFromQuery(r)
	completedStr := r.URL.Query().Get("completed")
	domain := r.URL.Query().Get("domain")

	query := "SELECT id, user_id, title, description, domain, screenshot_path, completed, created_at, updated_at FROM todos WHERE 1=1"
	args := []interface{}{}

	if userID > 0 {
		query += " AND user_id = ?"
		args = append(args, userID)
	}

	if completedStr != "" {
		completed, _ := strconv.ParseBool(completedStr)
		query += " AND completed = ?"
		args = append(args, completed)
	}

	if domain != "" {
		query += " AND domain = ?"
		args = append(args, domain)
	}

	rows, err := db.TodoDB.Query(query, args...)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	todos := []models.Todo{}
	for rows.Next() {
		var todo models.Todo
		err := rows.Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)
		if err != nil {
			continue
		}
		todos = append(todos, todo)
	}

	json.NewEncoder(w).Encode(todos)
}

func CreateTodo(w http.ResponseWriter, r *http.Request) {
	var todo models.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", todo.UserID)
	v.Required("title", todo.Title)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	result, err := db.TodoDB.Exec("INSERT INTO todos (user_id, title, description, domain, completed) VALUES (?, ?, ?, ?, ?)",
		todo.UserID, todo.Title, todo.Description, todo.Domain, todo.Completed)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	id, _ := result.LastInsertId()
	todo.ID = int(id)
	todo.CreatedAt = time.Now()
	todo.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

func GetTodoByID(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var todo models.Todo
	err := db.TodoDB.QueryRow("SELECT id, user_id, title, description, domain, screenshot_path, completed, created_at, updated_at FROM todos WHERE id = ?", id).
		Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Description, &todo.Domain, &todo.ScreenshotPath, &todo.Completed, &todo.CreatedAt, &todo.UpdatedAt)

	if err == sql.ErrNoRows {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	} else if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	var todo models.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	v := helpers.NewValidator()
	v.PositiveID("user_id", todo.UserID)
	v.Required("title", todo.Title)
	if !v.OK() {
		helpers.WriteValidationError(w, v.Fields())
		return
	}

	_, err := db.TodoDB.Exec("UPDATE todos SET user_id = ?, title = ?, description = ?, domain = ?, screenshot_path = ?, completed = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		todo.UserID, todo.Title, todo.Description, todo.Domain, todo.ScreenshotPath, todo.Completed, id)

	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	todo.ID = id
	todo.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id := helpers.GetIDFromPath(r)

	result, err := db.TodoDB.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		helpers.WriteError(w, http.StatusNotFound, "Todo not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
