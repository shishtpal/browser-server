package handlers

import (
	"encoding/json"
	"net/http"

	"browser-server/internal/models"
)

func GetRoutes(w http.ResponseWriter, r *http.Request) {
	routes := []models.Route{
		{Method: "POST", Path: "/routes", Description: "List all available routes"},
		{Method: "GET", Path: "/todos", Description: "Get all todos (filter: user_id, completed)"},
		{Method: "POST", Path: "/todos", Description: "Create a new todo"},
		{Method: "GET", Path: "/todos/{id}", Description: "Get todo by ID"},
		{Method: "PUT", Path: "/todos/{id}", Description: "Update todo by ID"},
		{Method: "DELETE", Path: "/todos/{id}", Description: "Delete todo by ID"},
		{Method: "GET", Path: "/bookmarks", Description: "Get all bookmarks (filter: user_id, tags)"},
		{Method: "POST", Path: "/bookmarks", Description: "Create a new bookmark"},
		{Method: "GET", Path: "/bookmarks/{id}", Description: "Get bookmark by ID"},
		{Method: "PUT", Path: "/bookmarks/{id}", Description: "Update bookmark by ID"},
		{Method: "DELETE", Path: "/bookmarks/{id}", Description: "Delete bookmark by ID"},
		{Method: "GET", Path: "/history", Description: "Get browsing history (filter: user_id, url)"},
		{Method: "POST", Path: "/history", Description: "Add history entry"},
		{Method: "GET", Path: "/history/{id}", Description: "Get history entry by ID"},
		{Method: "DELETE", Path: "/history/{id}", Description: "Delete history entry by ID"},
		{Method: "GET", Path: "/wallet", Description: "Get wallet entries (filter: user_id, website)"},
		{Method: "POST", Path: "/wallet", Description: "Create wallet entry"},
		{Method: "GET", Path: "/wallet/{id}", Description: "Get wallet entry by ID"},
		{Method: "PUT", Path: "/wallet/{id}", Description: "Update wallet entry by ID"},
		{Method: "DELETE", Path: "/wallet/{id}", Description: "Delete wallet entry by ID"},
		{Method: "GET", Path: "/users", Description: "Get all users"},
		{Method: "POST", Path: "/users", Description: "Create a new user"},
		{Method: "GET", Path: "/users/{id}", Description: "Get user by ID"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}
