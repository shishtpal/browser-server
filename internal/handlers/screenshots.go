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
		helpers.WriteError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	todoIDStr := r.URL.Query().Get("todo_id")
	if todoIDStr == "" {
		helpers.WriteError(w, http.StatusBadRequest, "Missing todo_id")
		return
	}
	todoID, err := strconv.Atoi(todoIDStr)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid todo_id")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Missing file")
		return
	}
	defer file.Close()

	var buf [16]byte
	rand.Read(buf[:])
	filename := fmt.Sprintf("%x.png", buf)

	outPath := filepath.Join(screenshotDir(), filename)
	out, err := os.Create(outPath)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to save file")
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	result, err := db.ScreenshotDB.Exec("INSERT INTO screenshots (todo_id, filename) VALUES (?, ?)", todoID, filename)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Database error")
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
		helpers.WriteError(w, http.StatusNotFound, "Screenshot not found")
		return
	}

	filePath := filepath.Join(screenshotDir(), screenshot.Filename)
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, filePath)
}
