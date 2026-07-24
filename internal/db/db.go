package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"browser-server/internal/helpers"
)

var (
	TodoDB       *sql.DB
	BookmarkDB   *sql.DB
	HistoryDB    *sql.DB
	WalletDB     *sql.DB
	UserDB       *sql.DB
	ScreenshotDB *sql.DB
	UsageDB      *sql.DB
)

func GetDataPath() string {
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		ex, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to get executable path:", err)
		}
		dataPath = filepath.Join(filepath.Dir(ex), ".data")
	}

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	return dataPath
}

func Open(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	return db
}

func Exec(db *sql.DB, query string) {
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to execute query:", err)
	}
}

func migrateColumn(db *sql.DB, table, column, colDef string) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&count)
	if err == nil && count == 0 {
		db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + colDef)
	}
}

func InitUserDB(dataPath string) {
	UserDB = Open(filepath.Join(dataPath, "users.db"))
	Exec(UserDB, `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL
		)
	`)
}

func InitTodoDB(dataPath string) {
	TodoDB = Open(filepath.Join(dataPath, "todos.db"))
	Exec(TodoDB, `
		CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			domain TEXT DEFAULT '',
			capture_id TEXT,
			screenshot_path TEXT DEFAULT '',
			completed BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	// migration: add new columns to existing databases
	migrateColumn(TodoDB, "todos", "domain", "TEXT DEFAULT ''")
	migrateColumn(TodoDB, "todos", "capture_id", "TEXT")
	migrateColumn(TodoDB, "todos", "screenshot_path", "TEXT DEFAULT ''")
	migrateColumn(TodoDB, "todos", "priority", "TEXT DEFAULT 'medium'")
	migrateColumn(TodoDB, "todos", "due_date", "DATETIME")
	migrateColumn(TodoDB, "todos", "tags", "TEXT DEFAULT '[]'")
	migrateColumn(TodoDB, "todos", "parent_id", "INTEGER")
	migrateColumn(TodoDB, "todos", "position", "INTEGER DEFAULT 0")
	Exec(TodoDB, `CREATE UNIQUE INDEX IF NOT EXISTS idx_todos_user_capture ON todos(user_id, capture_id)`)
	Exec(TodoDB, `CREATE INDEX IF NOT EXISTS idx_todos_parent ON todos(parent_id)`)
	Exec(TodoDB, `CREATE INDEX IF NOT EXISTS idx_todos_user_parent ON todos(user_id, parent_id, position)`)
}

func InitBookmarkDB(dataPath string) {
	BookmarkDB = Open(filepath.Join(dataPath, "bookmarks.db"))
	Exec(BookmarkDB, `
		CREATE TABLE IF NOT EXISTS bookmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			description TEXT,
			tags TEXT DEFAULT '[]',
			folder_path TEXT DEFAULT '',
			capture_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	migrateColumn(BookmarkDB, "bookmarks", "capture_id", "TEXT")
	Exec(BookmarkDB, `CREATE UNIQUE INDEX IF NOT EXISTS idx_bookmarks_user_capture ON bookmarks(user_id, capture_id)`)
}

func InitHistoryDB(dataPath string) {
	HistoryDB = Open(filepath.Join(dataPath, "history.db"))
	Exec(HistoryDB, `
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			domain TEXT NOT NULL DEFAULT '',
			title TEXT,
			visited_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			duration INTEGER DEFAULT 0
		)
	`)
	migrateColumn(HistoryDB, "history", "domain", "TEXT NOT NULL DEFAULT ''")
	backfillHistoryDomains()
	Exec(HistoryDB, `CREATE INDEX IF NOT EXISTS idx_history_user_domain ON history(user_id, domain)`)
}

func backfillHistoryDomains() {
	rows, err := HistoryDB.Query("SELECT id, url FROM history WHERE domain = ''")
	if err != nil {
		return
	}
	type update struct {
		id     int
		domain string
	}
	updates := []update{}
	for rows.Next() {
		var id int
		var rawURL string
		if err := rows.Scan(&id, &rawURL); err == nil {
			if domain := helpers.URLHostname(rawURL); domain != "" {
				updates = append(updates, update{id: id, domain: domain})
			}
		}
	}
	rows.Close()
	tx, err := HistoryDB.Begin()
	if err != nil {
		return
	}
	for _, item := range updates {
		if _, err := tx.Exec("UPDATE history SET domain = ? WHERE id = ?", item.domain, item.id); err != nil {
			log.Printf("Failed to backfill history domain: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit history domain backfill: %v", err)
	}
}

func InitScreenshotDB(dataPath string) {
	ScreenshotDB = Open(filepath.Join(dataPath, "screenshots.db"))
	screenshotDir := filepath.Join(dataPath, "screenshots")
	os.MkdirAll(screenshotDir, 0755)
	Exec(ScreenshotDB, `
		CREATE TABLE IF NOT EXISTS screenshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			todo_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			capture_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	migrateColumn(ScreenshotDB, "screenshots", "capture_id", "TEXT")
	Exec(ScreenshotDB, `CREATE UNIQUE INDEX IF NOT EXISTS idx_screenshots_todo_capture ON screenshots(todo_id, capture_id)`)
}

func InitWalletDB(dataPath string) {
	WalletDB = Open(filepath.Join(dataPath, "wallet.db"))
	Exec(WalletDB, `
		CREATE TABLE IF NOT EXISTS wallet (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			website TEXT NOT NULL,
			login_provider TEXT NOT NULL DEFAULT 'Password',
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	migrateColumn(WalletDB, "wallet", "login_provider", "TEXT NOT NULL DEFAULT 'Password'")
}

func InitUsageDB(dataPath string) {
	UsageDB = Open(filepath.Join(dataPath, "usage.db"))
	Exec(UsageDB, `
		CREATE TABLE IF NOT EXISTS domain_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			domain TEXT NOT NULL,
			date TEXT NOT NULL,
			total_seconds INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, domain, date)
		)
	`)
	Exec(UsageDB, `CREATE INDEX IF NOT EXISTS idx_domain_usage_user_date ON domain_usage(user_id, date)`)
	Exec(UsageDB, `CREATE INDEX IF NOT EXISTS idx_domain_usage_user_domain ON domain_usage(user_id, domain)`)
}

func InsertSampleData() {
	var count int

	err := UserDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err == nil && count == 0 {
		_, err = UserDB.Exec("INSERT INTO users (username, email) VALUES (?, ?)", "admin", "admin@example.com")
		if err != nil {
			log.Printf("Failed to insert sample user: %v", err)
		}
	}

	err = TodoDB.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count)
	if err == nil && count == 0 {
		_, err = TodoDB.Exec("INSERT INTO todos (user_id, title, description, domain, completed) VALUES (?, ?, ?, ?, ?)",
			1, "Sample Todo", "This is a sample todo", "", false)
		if err != nil {
			log.Printf("Failed to insert sample todo: %v", err)
		}
	}
}

func InitAll(dataPath string) {
	InitUserDB(dataPath)
	InitTodoDB(dataPath)
	InitBookmarkDB(dataPath)
	InitHistoryDB(dataPath)
	InitWalletDB(dataPath)
	InitScreenshotDB(dataPath)
	InitUsageDB(dataPath)
	InsertSampleData()
}

func CloseAll() {
	if TodoDB != nil {
		TodoDB.Close()
	}
	if BookmarkDB != nil {
		BookmarkDB.Close()
	}
	if HistoryDB != nil {
		HistoryDB.Close()
	}
	if WalletDB != nil {
		WalletDB.Close()
	}
	if UserDB != nil {
		UserDB.Close()
	}
	if ScreenshotDB != nil {
		ScreenshotDB.Close()
	}
	if UsageDB != nil {
		UsageDB.Close()
	}
}
