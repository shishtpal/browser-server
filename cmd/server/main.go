package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"

	"browser-server/internal/db"
	"browser-server/internal/handlers"
)

func main() {
	dataPath := db.GetDataPath()
	log.Printf("Using data path: %s", dataPath)

	db.InitAll(dataPath)
	defer db.CloseAll()

	r := mux.NewRouter()

	r.HandleFunc("/routes", handlers.GetRoutes).Methods("POST")

	r.HandleFunc("/todos", handlers.GetTodos).Methods("GET")
	r.HandleFunc("/todos", handlers.CreateTodo).Methods("POST")
	r.HandleFunc("/todos/{id}", handlers.GetTodoByID).Methods("GET")
	r.HandleFunc("/todos/{id}", handlers.UpdateTodo).Methods("PUT")
	r.HandleFunc("/todos/{id}", handlers.DeleteTodo).Methods("DELETE")

	r.HandleFunc("/bookmarks", handlers.GetBookmarks).Methods("GET")
	r.HandleFunc("/bookmarks", handlers.CreateBookmark).Methods("POST")
	r.HandleFunc("/bookmarks/{id}", handlers.GetBookmarkByID).Methods("GET")
	r.HandleFunc("/bookmarks/{id}", handlers.UpdateBookmark).Methods("PUT")
	r.HandleFunc("/bookmarks/{id}", handlers.DeleteBookmark).Methods("DELETE")

	r.HandleFunc("/history", handlers.GetHistory).Methods("GET")
	r.HandleFunc("/history", handlers.CreateHistory).Methods("POST")
	r.HandleFunc("/history/{id}", handlers.GetHistoryByID).Methods("GET")
	r.HandleFunc("/history/{id}", handlers.DeleteHistory).Methods("DELETE")

	r.HandleFunc("/wallet", handlers.GetWallet).Methods("GET")
	r.HandleFunc("/wallet", handlers.CreateWalletEntry).Methods("POST")
	r.HandleFunc("/wallet/{id}", handlers.GetWalletByID).Methods("GET")
	r.HandleFunc("/wallet/{id}", handlers.UpdateWalletEntry).Methods("PUT")
	r.HandleFunc("/wallet/{id}", handlers.DeleteWalletEntry).Methods("DELETE")

	r.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	r.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	r.HandleFunc("/users/{id}", handlers.GetUserByID).Methods("GET")

	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	staticFileDir := filepath.Join(filepath.Dir(ex), "frontend", "dist")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(staticFileDir))))

	fmt.Printf("Server starting on :8080\n")
	fmt.Printf("Database files location: %s\n", dataPath)
	fmt.Printf("Available routes:\n")
	fmt.Printf("POST /routes - List all routes\n")
	fmt.Printf("Multi-user API endpoints for todos, bookmarks, history, and wallet\n")
	fmt.Printf("\nTo change database location, set DATA_PATH environment variable\n")
	fmt.Printf("Example: DATA_PATH=/path/to/data ./server\n")

	log.Fatal(http.ListenAndServe(":8080", r))
}
