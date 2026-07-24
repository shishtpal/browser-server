package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	aiapi "browser-server/internal/ai/api"
	"browser-server/internal/auth"
	"browser-server/internal/db"
	"browser-server/internal/handlers"
	"browser-server/internal/middleware"
)

const defaultPort = "9191"

func main() {
	// CLI subcommands (e.g. `server token generate`) run and exit before the
	// HTTP server starts.
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "token" {
		runCLI(args)
		return
	}

	port, err := resolveServerPort(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		printUsage()
		os.Exit(1)
	}

	dataPath := db.GetDataPath()
	log.Printf("Using data path: %s", dataPath)

	db.InitAll(dataPath)
	defer db.CloseAll()

	aiModule, err := aiapi.Init()
	if err != nil {
		log.Fatalf("Failed to initialize AI module: %v", err)
	}
	defer aiModule.Close()

	if err := auth.Load(); err != nil {
		if os.IsNotExist(err) {
			log.Printf("WARNING: no API token found. Run 'server token generate' to create one; all /api requests will return 503 until then.")
		} else {
			log.Printf("WARNING: failed to load API token: %v", err)
		}
	} else {
		log.Printf("API token loaded; /api routes require Authorization: Bearer <token>")
	}

	r := mux.NewRouter()

	r.Use(middleware.Logging)
	r.Use(middleware.CORS)

	handlers.StartedAt = time.Now()
	// /health stays public for Docker/CI checks.
	r.HandleFunc("/health", handlers.Health).Methods("GET")

	// All /api routes require a valid API token.
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.Auth)

	api.HandleFunc("/routes", handlers.GetRoutes).Methods("POST")
	api.HandleFunc("/search/omnibox", handlers.SearchOmnibox).Methods("GET")
	aiModule.Register(api)

	api.HandleFunc("/todos", handlers.GetTodos).Methods("GET")
	api.HandleFunc("/todos", handlers.CreateTodo).Methods("POST")
	api.HandleFunc("/todos/reorder", handlers.ReorderTodos).Methods("POST")
	api.HandleFunc("/todos/{id}", handlers.GetTodoByID).Methods("GET")
	api.HandleFunc("/todos/{id}", handlers.UpdateTodo).Methods("PUT")
	api.HandleFunc("/todos/{id}", handlers.DeleteTodo).Methods("DELETE")
	api.HandleFunc("/todos/{id}/subtasks", handlers.GetSubtasks).Methods("GET")
	api.HandleFunc("/todos/{id}/subtasks", handlers.CreateSubtask).Methods("POST")

	api.HandleFunc("/screenshots", handlers.UploadScreenshot).Methods("POST")
	api.HandleFunc("/screenshots/{id}", handlers.GetScreenshot).Methods("GET")

	api.HandleFunc("/bookmarks", handlers.GetBookmarks).Methods("GET")
	api.HandleFunc("/bookmarks", handlers.CreateBookmark).Methods("POST")
	api.HandleFunc("/bookmarks/{id}", handlers.GetBookmarkByID).Methods("GET")
	api.HandleFunc("/bookmarks/{id}", handlers.UpdateBookmark).Methods("PUT")
	api.HandleFunc("/bookmarks/{id}", handlers.DeleteBookmark).Methods("DELETE")
	api.HandleFunc("/bookmarks/import", handlers.ImportBookmarks).Methods("POST")

	api.HandleFunc("/history", handlers.GetHistory).Methods("GET")
	api.HandleFunc("/history", handlers.CreateHistory).Methods("POST")
	api.HandleFunc("/history/grouped", handlers.GetGroupedHistory).Methods("GET")
	api.HandleFunc("/history/domains", handlers.GetHistoryDomains).Methods("GET")
	api.HandleFunc("/history/import", handlers.ImportHistory).Methods("POST")
	api.HandleFunc("/history/{id}", handlers.GetHistoryByID).Methods("GET")
	api.HandleFunc("/history/{id}", handlers.DeleteHistory).Methods("DELETE")

	api.HandleFunc("/analytics/usage", handlers.BatchUpsertUsage).Methods("POST")
	api.HandleFunc("/analytics/summary", handlers.GetAnalyticsSummary).Methods("GET")

	api.HandleFunc("/wallet", handlers.GetWallet).Methods("GET")
	api.HandleFunc("/wallet", handlers.CreateWalletEntry).Methods("POST")
	api.HandleFunc("/wallet/reveal", handlers.RevealWalletPassword).Methods("GET")
	api.HandleFunc("/wallet/import", handlers.ImportWallet).Methods("POST")
	api.HandleFunc("/wallet/{id}", handlers.GetWalletByID).Methods("GET")
	api.HandleFunc("/wallet/{id}", handlers.UpdateWalletEntry).Methods("PUT")
	api.HandleFunc("/wallet/{id}", handlers.DeleteWalletEntry).Methods("DELETE")

	api.HandleFunc("/users", handlers.GetUsers).Methods("GET")
	api.HandleFunc("/users", handlers.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", handlers.GetUserByID).Methods("GET")
	api.HandleFunc("/users/{id}", handlers.DeleteUser).Methods("DELETE")

	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	staticFileDir := filepath.Join(filepath.Dir(ex), "frontend", "dist")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir(staticFileDir))))

	fmt.Printf("Server starting on localhost:%s\n", port)
	fmt.Printf("Database files location: %s\n", dataPath)
	fmt.Printf("Available routes:\n")
	fmt.Printf("POST /api/routes - List all routes\n")
	fmt.Printf("Multi-user API endpoints under /api/ for todos, bookmarks, history, wallet, and users\n")
	fmt.Printf("\nTo change database location, set DATA_PATH environment variable\n")
	fmt.Printf("Example: DATA_PATH=/path/to/data ./server\n")
	fmt.Printf("\nTo change server port, pass --port or set PORT environment variable\n")
	fmt.Printf("Example: PORT=9090 ./server\n")

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func resolveServerPort(args []string) (string, error) {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	portFlag := flags.String("port", "", "HTTP server port")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("unknown argument: %s", flags.Arg(0))
	}

	port := os.Getenv("PORT")
	if *portFlag != "" {
		port = *portFlag
	}
	if port == "" {
		port = defaultPort
	}

	return validatePort(port)
}

func validatePort(port string) (string, error) {
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("invalid port %q: must be an integer", port)
	}
	if portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("invalid port %q: must be between 1 and 65535", port)
	}
	return strconv.Itoa(portNumber), nil
}

// runCLI handles non-server subcommands. Currently:
//
//	server token generate   - create and save a new API token (won't overwrite)
//	server token refresh    - regenerate and overwrite the API token
func runCLI(args []string) {
	switch args[0] {
	case "token":
		runTokenCLI(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func runTokenCLI(args []string) {
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "generate":
		token, path, err := auth.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("API token generated and saved to %s\n\n  %s\n\nSet this token in the web UI and browser extension to authenticate.\n", path, token)
	case "refresh":
		token, path, err := auth.Refresh()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("API token refreshed and saved to %s\n\n  %s\n\nUpdate the token in the web UI and browser extension to keep access.\n", path, token)
	default:
		fmt.Fprintf(os.Stderr, "unknown token command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  server [--port PORT]    Start the HTTP server\n")
	fmt.Fprintf(os.Stderr, "  server token generate   Generate and save a new API token\n")
	fmt.Fprintf(os.Stderr, "  server token refresh    Regenerate (rotate) the API token\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
	fmt.Fprintf(os.Stderr, "  PORT=9090 server        Start the HTTP server on port 9090\n")
}
