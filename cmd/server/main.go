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

	r.HandleFunc("/api/routes", handlers.GetRoutes).Methods("POST")

	r.HandleFunc("/api/todos", handlers.GetTodos).Methods("GET")
	r.HandleFunc("/api/todos", handlers.CreateTodo).Methods("POST")
	r.HandleFunc("/api/todos/{id}", handlers.GetTodoByID).Methods("GET")
	r.HandleFunc("/api/todos/{id}", handlers.UpdateTodo).Methods("PUT")
	r.HandleFunc("/api/todos/{id}", handlers.DeleteTodo).Methods("DELETE")

	r.HandleFunc("/api/screenshots", handlers.UploadScreenshot).Methods("POST")
	r.HandleFunc("/api/screenshots/{id}", handlers.GetScreenshot).Methods("GET")

	r.HandleFunc("/api/bookmarks", handlers.GetBookmarks).Methods("GET")
	r.HandleFunc("/api/bookmarks", handlers.CreateBookmark).Methods("POST")
	r.HandleFunc("/api/bookmarks/{id}", handlers.GetBookmarkByID).Methods("GET")
	r.HandleFunc("/api/bookmarks/{id}", handlers.UpdateBookmark).Methods("PUT")
	r.HandleFunc("/api/bookmarks/{id}", handlers.DeleteBookmark).Methods("DELETE")
	r.HandleFunc("/api/bookmarks/import", handlers.ImportBookmarks).Methods("POST")

	r.HandleFunc("/api/history", handlers.GetHistory).Methods("GET")
	r.HandleFunc("/api/history", handlers.CreateHistory).Methods("POST")
	r.HandleFunc("/api/history/import", handlers.ImportHistory).Methods("POST")
	r.HandleFunc("/api/history/{id}", handlers.GetHistoryByID).Methods("GET")
	r.HandleFunc("/api/history/{id}", handlers.DeleteHistory).Methods("DELETE")

	r.HandleFunc("/api/wallet", handlers.GetWallet).Methods("GET")
	r.HandleFunc("/api/wallet", handlers.CreateWalletEntry).Methods("POST")
	r.HandleFunc("/api/wallet/{id}", handlers.GetWalletByID).Methods("GET")
	r.HandleFunc("/api/wallet/{id}", handlers.UpdateWalletEntry).Methods("PUT")
	r.HandleFunc("/api/wallet/{id}", handlers.DeleteWalletEntry).Methods("DELETE")

	r.HandleFunc("/api/users", handlers.GetUsers).Methods("GET")
	r.HandleFunc("/api/users", handlers.CreateUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", handlers.GetUserByID).Methods("GET")
	r.HandleFunc("/api/users/{id}", handlers.DeleteUser).Methods("DELETE")

	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	staticFileDir := filepath.Join(filepath.Dir(ex), "frontend", "dist")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(staticFileDir))))

	fmt.Printf("Server starting on localhost:8080\n")
	fmt.Printf("Database files location: %s\n", dataPath)
	fmt.Printf("Available routes:\n")
	fmt.Printf("POST /api/routes - List all routes\n")
	fmt.Printf("Multi-user API endpoints under /api/ for todos, bookmarks, history, wallet, and users\n")
	fmt.Printf("\nTo change database location, set DATA_PATH environment variable\n")
	fmt.Printf("Example: DATA_PATH=/path/to/data ./server\n")

	log.Fatal(http.ListenAndServe(":8080", r))
}
