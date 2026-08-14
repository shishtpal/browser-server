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
	"browser-server/internal/quiz"
	quizconfig "browser-server/internal/quiz/config"
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
	handlers.AdminShutdown = aiModule.PrepareRestart

	// The quiz feature is fully gated by bs-quiz-config.json: when the file is
	// missing or enabled is false, no quiz database is created and no routes
	// are registered.
	quizCfg, err := quizconfig.Load()
	if err != nil {
		log.Fatalf("Failed to load quiz config: %v", err)
	}

	if err := auth.Load(); err != nil {
		if os.IsNotExist(err) {
			log.Printf("WARNING: no API token found. Run 'server token generate' to create one; operator /api requests will return 503 until then.")
		} else {
			log.Printf("WARNING: failed to load API token: %v", err)
		}
	} else {
		log.Printf("API token loaded; operator API routes require Authorization: Bearer <token>")
	}
	if err := auth.AdminLoad(); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Admin API disabled; run 'server token admin-generate' and restart to enable it")
		} else {
			log.Printf("WARNING: failed to load admin API token: %v", err)
		}
	} else {
		log.Printf("Admin API enabled with a separate administrator token")
	}

	r := mux.NewRouter()

	r.Use(middleware.Logging)
	r.Use(middleware.CORS(aiModule.CORSEnabled()))

	handlers.StartedAt = time.Now()
	handlers.ServerPort = port
	// /health stays public for Docker/CI checks and restart polling.
	r.HandleFunc("/health", handlers.Health).Methods("GET")

	// Register the more-specific admin prefix before the general /api prefix:
	// gorilla/mux evaluates routes in order. Admin routes accept only the
	// separate administrator token, not the day-to-day operator token.
	admin := r.PathPrefix("/api/admin").Subrouter()
	admin.Use(middleware.AdminAuth)
	aiModule.RegisterAdmin(admin)
	admin.HandleFunc("/restart", handlers.AdminRestart).Methods(http.MethodPost)
	admin.HandleFunc("/status", handlers.AdminStatus).Methods(http.MethodGet)

	// All remaining /api routes require a valid operator token.
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

	api.HandleFunc("/prompts", handlers.GetPrompts).Methods("GET")
	api.HandleFunc("/prompts", handlers.CreatePrompt).Methods("POST")
	api.HandleFunc("/prompts/search", handlers.SearchPrompts).Methods("GET")
	api.HandleFunc("/prompts/{id:[0-9]+}", handlers.GetPromptByID).Methods("GET")
	api.HandleFunc("/prompts/{id:[0-9]+}", handlers.UpdatePrompt).Methods("PUT")
	api.HandleFunc("/prompts/{id:[0-9]+}", handlers.DeletePrompt).Methods("DELETE")

	if quizCfg.Enabled {
		quiz.SetDefaultScheduler(quizCfg.Scheduler)
		db.InitQuizDB(dataPath)
		defer db.CloseQuizDB()
		if err := os.MkdirAll(quizCfg.ResolveImageDir(dataPath), 0755); err != nil {
			log.Fatalf("Failed to create quiz image dir: %v", err)
		}
		api.HandleFunc("/quiz/questions", handlers.GetQuestions).Methods("GET")
		api.HandleFunc("/quiz/questions", handlers.CreateQuestion).Methods("POST")
		api.HandleFunc("/quiz/questions/{id:[0-9]+}", handlers.GetQuestionByID).Methods("GET")
		api.HandleFunc("/quiz/questions/{id:[0-9]+}", handlers.UpdateQuestion).Methods("PUT")
		api.HandleFunc("/quiz/questions/{id:[0-9]+}", handlers.DeleteQuestion).Methods("DELETE")
		api.HandleFunc("/quiz/questions/{id:[0-9]+}/image", handlers.UploadQuestionImage).Methods("POST")
		api.HandleFunc("/quiz/questions/{id:[0-9]+}/image", handlers.GetQuestionImage).Methods("GET")
		api.HandleFunc("/quiz/cards", handlers.GetQuestionCards).Methods("GET")
		api.HandleFunc("/quiz/cards/{id:[0-9]+}/review", handlers.ReviewQuestionCard).Methods("POST")
		api.HandleFunc("/quiz/cards/{id:[0-9]+}/skip", handlers.SkipQuestionCard).Methods("POST")
		api.HandleFunc("/quiz/papers", handlers.GeneratePaper).Methods("POST")
		api.HandleFunc("/quiz/papers", handlers.GetPapers).Methods("GET")
		api.HandleFunc("/quiz/papers/{id:[0-9]+}", handlers.GetPaperByID).Methods("GET")
		api.HandleFunc("/quiz/papers/{id:[0-9]+}", handlers.DeletePaper).Methods("DELETE")
		api.HandleFunc("/quiz/tags", handlers.GetTagVocabulary).Methods("GET")
		api.HandleFunc("/quiz/stats", handlers.GetQuizStats).Methods("GET")
		api.HandleFunc("/users/{id:[0-9]+}/quiz-settings", handlers.GetQuizSettings).Methods("GET")
		api.HandleFunc("/users/{id:[0-9]+}/quiz-settings", handlers.UpdateQuizSettings).Methods("POST")
		log.Printf("Quiz feature enabled (db: %s)", quizCfg.DBPath)
	}

	ex, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	staticFileDir := filepath.Join(filepath.Dir(ex), "frontend", "dist")

	// Conversation URLs are client-side state. Serve the chat shell for a direct
	// /chat/{conversation-id} visit so shared links and browser history work.
	r.PathPrefix("/chat/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(staticFileDir, "chat", "index.html"))
	}).Methods(http.MethodGet, http.MethodHead)

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

// runCLI handles non-server token-management subcommands for both the
// operator and administrator credential tiers.
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
	case "admin-generate":
		token, path, err := auth.AdminGenerate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Admin API token generated and saved to %s\n\n  %s\n\nRestart the server, then save this separate token on the Project Settings page.\n", path, token)
	case "admin-refresh":
		token, path, err := auth.AdminRefresh()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Admin API token refreshed and saved to %s\n\n  %s\n\nRestart the server and update the Project Settings page.\n", path, token)
	case "admin-delete":
		path, err := auth.AdminDelete()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Admin API token removed from %s. Restart the server to disable the admin API in the running process.\n", path)
	default:
		fmt.Fprintf(os.Stderr, "unknown token command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  server [--port PORT]    Start the HTTP server\n")
	fmt.Fprintf(os.Stderr, "  server token generate        Generate and save a new operator API token\n")
	fmt.Fprintf(os.Stderr, "  server token refresh         Rotate the operator API token\n")
	fmt.Fprintf(os.Stderr, "  server token admin-generate  Generate the separate admin API token\n")
	fmt.Fprintf(os.Stderr, "  server token admin-refresh   Rotate the admin API token\n")
	fmt.Fprintf(os.Stderr, "  server token admin-delete    Remove and disable the admin token on restart\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
	fmt.Fprintf(os.Stderr, "  PORT=9090 server             Start the HTTP server on port 9090\n")
	fmt.Fprintf(os.Stderr, "  SERVER_ADMIN_TOKEN_PATH=...  Override the admin token file path\n")
}
