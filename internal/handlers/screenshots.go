package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"browser-server/internal/db"
	"browser-server/internal/helpers"
	"browser-server/internal/models"
)

func screenshotDir() string {
	return filepath.Join(db.GetDataPath(), "screenshots")
}

func UploadScreenshot(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	todoIDStr := r.URL.Query().Get("todo_id")
	if todoIDStr == "" {
		http.Error(w, "Missing todo_id", http.StatusBadRequest)
		return
	}
	todoID, err := strconv.Atoi(todoIDStr)
	if err != nil {
		http.Error(w, "Invalid todo_id", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf [16]byte
	rand.Read(buf[:])
	filename := fmt.Sprintf("%x.png", buf)

	outPath := filepath.Join(screenshotDir(), filename)
	out, err := os.Create(outPath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	result, err := db.ScreenshotDB.Exec("INSERT INTO screenshots (todo_id, filename) VALUES (?, ?)", todoID, filename)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	screenshotID, _ := result.LastInsertId()

	db.TodoDB.Exec("UPDATE todos SET screenshot_path = ? WHERE id = ?", filename, todoID)

	screenshot := models.Screenshot{
		ID:       int(screenshotID),
		TodoID:   todoID,
		Filename: filename,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(screenshot)
}

func GetScreenshot(w http.ResponseWriter, r *http.Request) {
	todoID := helpers.GetIDFromPath(r)

	var screenshot models.Screenshot
	err := db.ScreenshotDB.QueryRow("SELECT id, todo_id, filename, created_at FROM screenshots WHERE todo_id = ? ORDER BY id DESC LIMIT 1", todoID).
		Scan(&screenshot.ID, &screenshot.TodoID, &screenshot.Filename, &screenshot.CreatedAt)
	if err != nil {
		http.Error(w, "Screenshot not found", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(screenshotDir(), screenshot.Filename)
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, filePath)
}
