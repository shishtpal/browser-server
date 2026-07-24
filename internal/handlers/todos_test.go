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
		INSERT INTO todos (user_id, title, description, domain, screenshot_path, completed, priority, tags, position)
		VALUES (1, 'Keep title', 'Keep description', 'example.com', 'todo.png', 1, 'medium', '["work"]', 7)`)
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
	if len(updated.Tags) != 1 || updated.Tags[0] != "work" {
		t.Fatalf("partial update changed tags: %+v", updated.Tags)
	}
}
