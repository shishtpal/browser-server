package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var (
	TodoDB     *sql.DB
	BookmarkDB *sql.DB
	HistoryDB  *sql.DB
	WalletDB   *sql.DB
	UserDB     *sql.DB
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
			completed BOOLEAN DEFAULT FALSE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
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
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
}

func InitHistoryDB(dataPath string) {
	HistoryDB = Open(filepath.Join(dataPath, "history.db"))
	Exec(HistoryDB, `
		CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			title TEXT,
			visited_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			duration INTEGER DEFAULT 0
		)
	`)
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
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
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
		_, err = TodoDB.Exec("INSERT INTO todos (user_id, title, description, completed) VALUES (?, ?, ?, ?)",
			1, "Sample Todo", "This is a sample todo", false)
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
}
