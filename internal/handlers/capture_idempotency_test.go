package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"browser-server/internal/db"
	"browser-server/internal/models"
)

func postJSON(t *testing.T, handler http.HandlerFunc, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler(response, request)
	return response
}

func uploadTestScreenshot(t *testing.T, todoID int, captureID string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "capture.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := file.Write([]byte("png-data")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/screenshots?todo_id="+strconv.Itoa(todoID)+"&capture_id="+captureID,
		&body,
	)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	UploadScreenshot(response, request)
	return response
}

func TestCaptureRetriesAreIdempotent(t *testing.T) {
	dataPath := t.TempDir()
	t.Setenv("DATA_PATH", dataPath)
	db.InitAll(dataPath)
	t.Cleanup(db.CloseAll)

	todoInput := models.Todo{
		UserID:      1,
		Title:       "Captured page",
		Description: "Source: https://example.com/page",
		Domain:      "example.com",
		CaptureID:   "todo-capture-1",
	}
	firstTodoResponse := postJSON(t, CreateTodo, "/api/todos", todoInput)
	if firstTodoResponse.Code != http.StatusCreated {
		t.Fatalf("first todo status = %d, want %d: %s", firstTodoResponse.Code, http.StatusCreated, firstTodoResponse.Body.String())
	}
	var firstTodo models.TodoResponse
	if err := json.NewDecoder(firstTodoResponse.Body).Decode(&firstTodo); err != nil {
		t.Fatalf("decode first todo: %v", err)
	}

	secondTodoResponse := postJSON(t, CreateTodo, "/api/todos", todoInput)
	if secondTodoResponse.Code != http.StatusOK {
		t.Fatalf("second todo status = %d, want %d: %s", secondTodoResponse.Code, http.StatusOK, secondTodoResponse.Body.String())
	}
	var secondTodo models.TodoResponse
	if err := json.NewDecoder(secondTodoResponse.Body).Decode(&secondTodo); err != nil {
		t.Fatalf("decode second todo: %v", err)
	}
	if secondTodo.ID != firstTodo.ID {
		t.Fatalf("retried todo ID = %d, want %d", secondTodo.ID, firstTodo.ID)
	}

	bookmarkInput := models.BookmarkResponse{
		UserID:      1,
		Title:       "Example",
		URL:         "https://example.com/page",
		Description: "Source: https://example.com/page",
		CaptureID:   "bookmark-capture-1",
	}
	firstBookmarkResponse := postJSON(t, CreateBookmark, "/api/bookmarks", bookmarkInput)
	if firstBookmarkResponse.Code != http.StatusCreated {
		t.Fatalf("first bookmark status = %d, want %d: %s", firstBookmarkResponse.Code, http.StatusCreated, firstBookmarkResponse.Body.String())
	}
	var firstBookmark models.BookmarkResponse
	if err := json.NewDecoder(firstBookmarkResponse.Body).Decode(&firstBookmark); err != nil {
		t.Fatalf("decode first bookmark: %v", err)
	}

	secondBookmarkResponse := postJSON(t, CreateBookmark, "/api/bookmarks", bookmarkInput)
	if secondBookmarkResponse.Code != http.StatusOK {
		t.Fatalf("second bookmark status = %d, want %d: %s", secondBookmarkResponse.Code, http.StatusOK, secondBookmarkResponse.Body.String())
	}
	var secondBookmark models.BookmarkResponse
	if err := json.NewDecoder(secondBookmarkResponse.Body).Decode(&secondBookmark); err != nil {
		t.Fatalf("decode second bookmark: %v", err)
	}
	if secondBookmark.ID != firstBookmark.ID {
		t.Fatalf("retried bookmark ID = %d, want %d", secondBookmark.ID, firstBookmark.ID)
	}

	firstScreenshotResponse := uploadTestScreenshot(t, firstTodo.ID, "todo-capture-1")
	if firstScreenshotResponse.Code != http.StatusCreated {
		t.Fatalf("first screenshot status = %d, want %d: %s", firstScreenshotResponse.Code, http.StatusCreated, firstScreenshotResponse.Body.String())
	}
	var firstScreenshot models.Screenshot
	if err := json.NewDecoder(firstScreenshotResponse.Body).Decode(&firstScreenshot); err != nil {
		t.Fatalf("decode first screenshot: %v", err)
	}
	if _, err := db.TodoDB.Exec("UPDATE todos SET screenshot_path = '' WHERE id = ?", firstTodo.ID); err != nil {
		t.Fatalf("clear screenshot path before retry: %v", err)
	}

	secondScreenshotResponse := uploadTestScreenshot(t, firstTodo.ID, "todo-capture-1")
	if secondScreenshotResponse.Code != http.StatusOK {
		t.Fatalf("second screenshot status = %d, want %d: %s", secondScreenshotResponse.Code, http.StatusOK, secondScreenshotResponse.Body.String())
	}
	var secondScreenshot models.Screenshot
	if err := json.NewDecoder(secondScreenshotResponse.Body).Decode(&secondScreenshot); err != nil {
		t.Fatalf("decode second screenshot: %v", err)
	}
	if secondScreenshot.ID != firstScreenshot.ID {
		t.Fatalf("retried screenshot ID = %d, want %d", secondScreenshot.ID, firstScreenshot.ID)
	}
	var repairedPath string
	if err := db.TodoDB.QueryRow("SELECT screenshot_path FROM todos WHERE id = ?", firstTodo.ID).Scan(&repairedPath); err != nil {
		t.Fatalf("read repaired screenshot path: %v", err)
	}
	if repairedPath != firstScreenshot.Filename {
		t.Fatalf("repaired screenshot path = %q, want %q", repairedPath, firstScreenshot.Filename)
	}

	missingTodoResponse := uploadTestScreenshot(t, firstTodo.ID+1000, "missing-todo-capture")
	if missingTodoResponse.Code != http.StatusNotFound {
		t.Fatalf("missing todo screenshot status = %d, want %d: %s", missingTodoResponse.Code, http.StatusNotFound, missingTodoResponse.Body.String())
	}

	var todoCount, bookmarkCount, screenshotCount int
	if err := db.TodoDB.QueryRow("SELECT COUNT(*) FROM todos WHERE capture_id = ?", todoInput.CaptureID).Scan(&todoCount); err != nil {
		t.Fatalf("count todos: %v", err)
	}
	if err := db.BookmarkDB.QueryRow("SELECT COUNT(*) FROM bookmarks WHERE capture_id = ?", bookmarkInput.CaptureID).Scan(&bookmarkCount); err != nil {
		t.Fatalf("count bookmarks: %v", err)
	}
	if err := db.ScreenshotDB.QueryRow("SELECT COUNT(*) FROM screenshots WHERE capture_id = ?", todoInput.CaptureID).Scan(&screenshotCount); err != nil {
		t.Fatalf("count screenshots: %v", err)
	}
	if todoCount != 1 || bookmarkCount != 1 || screenshotCount != 1 {
		t.Fatalf("capture row counts = todo %d, bookmark %d, screenshot %d; want all 1", todoCount, bookmarkCount, screenshotCount)
	}

	files, err := os.ReadDir(filepath.Join(dataPath, "screenshots"))
	if err != nil {
		t.Fatalf("read screenshots directory: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("screenshot file count = %d, want 1", len(files))
	}
}
