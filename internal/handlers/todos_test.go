package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/models"

	"github.com/gorilla/mux"
)

func TestUpdateTodoAllowsPartialPriorityUpdate(t *testing.T) {
	dataPath := t.TempDir()
	db.InitTodoDB(dataPath)
	t.Cleanup(func() { db.TodoDB.Close() })

	result, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, domain, screenshot_path, pinned, status, priority, tags, position)
		VALUES (1, 'Keep title', 'Keep description', 'example.com', 'todo.png', 1, 'completed', 'medium', '["work"]', 7)`)
	if err != nil {
		t.Fatalf("insert todo: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get todo ID: %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/todos/1", bytes.NewBufferString(`{"priority":"high"}`))
	request = mux.SetURLVars(request, map[string]string{"id": "1"})
	response := httptest.NewRecorder()
	UpdateTodo(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}

	var updated models.TodoResponse
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if updated.ID != int(id) || updated.Priority != "high" {
		t.Fatalf("unexpected updated todo: %+v", updated)
	}
	if updated.Title != "Keep title" || updated.Description != "Keep description" || updated.Domain != "example.com" {
		t.Fatalf("partial update changed text fields: %+v", updated)
	}
	if updated.Status != "completed" || updated.ScreenshotPath != "todo.png" || updated.Position != 7 {
		t.Fatalf("partial update changed other fields: %+v", updated)
	}
	if !updated.Pinned {
		t.Fatalf("partial update changed pinned state: %+v", updated)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "work" {
		t.Fatalf("partial update changed tags: %+v", updated.Tags)
	}
}

func TestGetTodosArchiveFilteringAndPinnedOrdering(t *testing.T) {
	dataPath := t.TempDir()
	db.InitTodoDB(dataPath)
	t.Cleanup(func() { db.TodoDB.Close() })

	_, err := db.TodoDB.Exec(`
		INSERT INTO todos (user_id, title, description, status, pinned, priority, tags, position) VALUES
		(1, 'active-unpinned', '', 'pending', 0, 'urgent', '[]', 1),
		(1, 'active-pinned', '', 'pending', 1, 'low', '[]', 99),
		(1, 'archived-todo', '', 'archived', 0, 'medium', '[]', 0)`)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/todos?sort=priority&order=asc", nil)
	response := httptest.NewRecorder()
	GetTodos(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("default get status %d: %s", response.Code, response.Body.String())
	}
	var active []models.TodoResponse
	if err := json.NewDecoder(response.Body).Decode(&active); err != nil {
		t.Fatalf("decode active todos: %v", err)
	}
	if len(active) != 2 || active[0].Title != "active-pinned" || !active[0].Pinned || active[1].Title != "active-unpinned" {
		t.Fatalf("expected only active todos with pinned first, got %+v", active)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/todos?archived=true", nil)
	response = httptest.NewRecorder()
	GetTodos(response, request)
	var all []models.TodoResponse
	if err := json.NewDecoder(response.Body).Decode(&all); err != nil {
		t.Fatalf("decode all todos: %v", err)
	}
	// With archived=true, all todos including archived are shown
	hasArchived := false
	for _, todo := range all {
		if todo.Status == "archived" {
			hasArchived = true
		}
	}
	if !hasArchived {
		t.Fatalf("expected archived todo when archived=true, got %+v", all)
	}
}

func TestUpdateTodoPinnedStatusPartialState(t *testing.T) {
	dataPath := t.TempDir()
	db.InitTodoDB(dataPath)
	t.Cleanup(func() { db.TodoDB.Close() })

	result, err := db.TodoDB.Exec("INSERT INTO todos (user_id, title, description, pinned, status, priority, tags) VALUES (1, 'state', '', 0, 'pending', 'medium', '[]')")
	if err != nil {
		t.Fatalf("insert todo: %v", err)
	}
	id, _ := result.LastInsertId()

	update := func(body string) models.TodoResponse {
		t.Helper()
		request := httptest.NewRequest(http.MethodPut, "/api/todos/1", bytes.NewBufferString(body))
		request = mux.SetURLVars(request, map[string]string{"id": "1"})
		response := httptest.NewRecorder()
		UpdateTodo(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("update status %d: %s", response.Code, response.Body.String())
		}
		var todo models.TodoResponse
		if err := json.NewDecoder(response.Body).Decode(&todo); err != nil {
			t.Fatalf("decode update: %v", err)
		}
		return todo
	}

	updated := update(`{"pinned":true,"status":"archived"}`)
	if updated.ID != int(id) || !updated.Pinned || updated.Status != "archived" {
		t.Fatalf("state fields not updated: %+v", updated)
	}
	updated = update(`{"title":"renamed"}`)
	if !updated.Pinned || updated.Status != "archived" || updated.Title != "renamed" {
		t.Fatalf("unrelated partial update did not preserve state: %+v", updated)
	}
}
