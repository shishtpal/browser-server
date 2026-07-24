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
		INSERT INTO todos (user_id, title, description, domain, screenshot_path, completed, pinned, archived, priority, tags, position)
		VALUES (1, 'Keep title', 'Keep description', 'example.com', 'todo.png', 1, 1, 1, 'medium', '["work"]', 7)`)
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
	if !updated.Completed || updated.ScreenshotPath != "todo.png" || updated.Position != 7 {
		t.Fatalf("partial update changed other fields: %+v", updated)
	}
	if !updated.Pinned || !updated.Archived {
		t.Fatalf("partial update changed pinned/archive state: %+v", updated)
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
		INSERT INTO todos (user_id, title, description, archived, pinned, priority, tags, position) VALUES
		(1, 'active-unpinned', '', 0, 0, 'urgent', '[]', 1),
		(1, 'active-pinned', '', 0, 1, 'low', '[]', 99),
		(1, 'archived', '', 1, 0, 'medium', '[]', 0)`)
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
	var archived []models.TodoResponse
	if err := json.NewDecoder(response.Body).Decode(&archived); err != nil {
		t.Fatalf("decode archived todos: %v", err)
	}
	if len(archived) != 1 || archived[0].Title != "archived" || !archived[0].Archived {
		t.Fatalf("expected only archived todo, got %+v", archived)
	}
}

func TestUpdateTodoPinnedArchivedPartialState(t *testing.T) {
	dataPath := t.TempDir()
	db.InitTodoDB(dataPath)
	t.Cleanup(func() { db.TodoDB.Close() })

	result, err := db.TodoDB.Exec("INSERT INTO todos (user_id, title, description, pinned, archived, priority, tags) VALUES (1, 'state', '', 0, 0, 'medium', '[]')")
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

	updated := update(`{"pinned":true,"archived":true}`)
	if updated.ID != int(id) || !updated.Pinned || !updated.Archived {
		t.Fatalf("state fields not updated: %+v", updated)
	}
	updated = update(`{"title":"renamed"}`)
	if !updated.Pinned || !updated.Archived || updated.Title != "renamed" {
		t.Fatalf("unrelated partial update did not preserve state: %+v", updated)
	}
}
