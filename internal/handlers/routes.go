package handlers

import (
	"encoding/json"
	"net/http"

	"browser-server/internal/models"
)

func GetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := []models.Route{
		{Method: "POST", Path: "/api/routes", Description: "List all available routes"},
		{Method: "GET", Path: "/api/todos", Description: "Get all todos (filter: user_id, domain, completed)"},
		{Method: "POST", Path: "/api/todos", Description: "Create a new todo"},
		{Method: "GET", Path: "/api/todos/{id}", Description: "Get todo by ID"},
		{Method: "PUT", Path: "/api/todos/{id}", Description: "Update todo by ID"},
		{Method: "DELETE", Path: "/api/todos/{id}", Description: "Delete todo by ID"},
		{Method: "POST", Path: "/api/screenshots", Description: "Upload a screenshot for a todo (multipart: file + ?todo_id=)"},
		{Method: "GET", Path: "/api/screenshots/{id}", Description: "Get screenshot PNG by todo ID"},
		{Method: "GET", Path: "/api/bookmarks", Description: "Get all bookmarks (filter: user_id, tags)"},
		{Method: "POST", Path: "/api/bookmarks", Description: "Create a new bookmark"},
		{Method: "GET", Path: "/api/bookmarks/{id}", Description: "Get bookmark by ID"},
		{Method: "PUT", Path: "/api/bookmarks/{id}", Description: "Update bookmark by ID"},
		{Method: "DELETE", Path: "/api/bookmarks/{id}", Description: "Delete bookmark by ID"},
		{Method: "POST", Path: "/api/bookmarks/import", Description: "Import bookmarks from Chrome HTML export"},


		{Method: "GET", Path: "/api/history", Description: "Get browsing history (filter: user_id, url)"},
		{Method: "POST", Path: "/api/history", Description: "Add history entry"},
		{Method: "POST", Path: "/api/history/import", Description: "Import history from Chrome History SQLite file"},
		{Method: "GET", Path: "/api/history/{id}", Description: "Get history entry by ID"},
		{Method: "DELETE", Path: "/api/history/{id}", Description: "Delete history entry by ID"},
		{Method: "GET", Path: "/api/wallet", Description: "Get wallet entries (filter: user_id, website)"},
		{Method: "POST", Path: "/api/wallet", Description: "Create wallet entry"},
		{Method: "GET", Path: "/api/wallet/{id}", Description: "Get wallet entry by ID"},
		{Method: "PUT", Path: "/api/wallet/{id}", Description: "Update wallet entry by ID"},
		{Method: "DELETE", Path: "/api/wallet/{id}", Description: "Delete wallet entry by ID"},
		{Method: "GET", Path: "/api/users", Description: "Get all users"},
		{Method: "POST", Path: "/api/users", Description: "Create a new user"},
		{Method: "GET", Path: "/api/users/{id}", Description: "Get user by ID"},
		{Method: "DELETE", Path: "/api/users/{id}", Description: "Delete user by ID"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}
