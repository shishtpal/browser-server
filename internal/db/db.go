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
	PromptDB     *sql.DB
	QuizDB       *sql.DB
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

// tryExec runs a migration statement, logging but not aborting on error.
// Used for best-effort column drops and other idempotent migrations that
// may legitimately fail on older SQLite versions.
func tryExec(db *sql.DB, query string) {
	if _, err := db.Exec(query); err != nil {
		log.Printf("Best-effort migration skipped: %v (query: %s)", err, query)
	}
}

func hasColumn(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}

func ensureColumn(db *sql.DB, tableName, columnName, definition string) {
	exists, err := hasColumn(db, tableName, columnName)
	if err != nil {
		log.Fatal("Failed to inspect schema:", err)
	}
	if exists {
		return
	}
	if _, err := db.Exec("ALTER TABLE " + tableName + " ADD COLUMN " + definition); err != nil {
		log.Fatal("Failed to add column:", err)
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
			description TEXT DEFAULT '',
			domain TEXT DEFAULT '',
			capture_id TEXT,
			screenshot_path TEXT DEFAULT '',
			pinned BOOLEAN DEFAULT FALSE,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'medium',
			color TEXT DEFAULT '',
			start_date DATETIME,
			end_date DATETIME,
			rrule TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			parent_id INTEGER,
			position INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	ensureColumn(TodoDB, "todos", "capture_id", "capture_id TEXT")
	ensureColumn(TodoDB, "todos", "color", "color TEXT DEFAULT ''")
	ensureColumn(TodoDB, "todos", "start_date", "start_date DATETIME")
	ensureColumn(TodoDB, "todos", "end_date", "end_date DATETIME")
	ensureColumn(TodoDB, "todos", "rrule", "rrule TEXT DEFAULT ''")
	ensureColumn(TodoDB, "todos", "parent_id", "parent_id INTEGER")
	ensureColumn(TodoDB, "todos", "position", "position INTEGER DEFAULT 0")
	ensureColumn(TodoDB, "todos", "created_at", "created_at DATETIME DEFAULT CURRENT_TIMESTAMP")
	ensureColumn(TodoDB, "todos", "updated_at", "updated_at DATETIME DEFAULT CURRENT_TIMESTAMP")
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

func InitPromptDB(dataPath string) {
	PromptDB = Open(filepath.Join(dataPath, "prompts.db"))
	Exec(PromptDB, `
		CREATE TABLE IF NOT EXISTS prompts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			description TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	Exec(PromptDB, `CREATE INDEX IF NOT EXISTS idx_prompts_user ON prompts(user_id)`)
}

func InitQuizDB(dataPath string) {
	QuizDB = Open(filepath.Join(dataPath, "quiz.db"))
	Exec(QuizDB, `
		CREATE TABLE IF NOT EXISTS questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			difficulty TEXT NOT NULL DEFAULT 'medium',
			question TEXT NOT NULL,
			explanation TEXT DEFAULT '',
			options_json TEXT DEFAULT '[]',
			answer_json TEXT DEFAULT '[]',
			image_filename TEXT DEFAULT '',
			tags TEXT DEFAULT '[]',
			subject TEXT DEFAULT '',
			topic TEXT DEFAULT '',
			sub_topic TEXT DEFAULT '',
			source TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user ON questions(user_id)`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user_type ON questions(user_id, type)`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user_difficulty ON questions(user_id, difficulty)`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user_subject ON questions(user_id, subject)`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user_topic ON questions(user_id, topic)`)

	// One-shot migration: replace the single `exam` column with a JSON-array
	// `tags` column. Idempotent — re-runs are a no-op.
	if hasExam, err := hasColumn(QuizDB, "questions", "exam"); err != nil {
		log.Fatal("Failed to inspect questions table:", err)
	} else if hasExam {
		// Migrate the existing single-value exam into a one-element tags
		// array so no rows lose their exam info if the operator hasn't
		// deleted the database yet.
		Exec(QuizDB, `ALTER TABLE questions ADD COLUMN tags TEXT DEFAULT '[]'`)
		Exec(QuizDB, `UPDATE questions SET tags = json_array(exam) WHERE exam != '' AND (tags IS NULL OR tags = '' OR tags = '[]')`)
		Exec(QuizDB, `DROP INDEX IF EXISTS idx_questions_user_exam`)
		// DROP COLUMN requires SQLite >= 3.35. Skip if the host's build is
		// older; the unused column is harmless.
		tryExec(QuizDB, `ALTER TABLE questions DROP COLUMN exam`)
	}
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_questions_user_tags ON questions(user_id, tags)`)

	Exec(QuizDB, `
		CREATE TABLE IF NOT EXISTS question_papers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			sections_json TEXT NOT NULL,
			question_count INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_papers_user ON question_papers(user_id)`)

	Exec(QuizDB, `
		CREATE TABLE IF NOT EXISTS question_paper_items (
			paper_id INTEGER NOT NULL,
			question_id INTEGER NOT NULL,
			position INTEGER NOT NULL,
			PRIMARY KEY (paper_id, question_id),
			FOREIGN KEY (paper_id) REFERENCES question_papers(id) ON DELETE CASCADE,
			FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
		)
	`)

	Exec(QuizDB, `
		CREATE TABLE IF NOT EXISTS question_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	Exec(QuizDB, `CREATE INDEX IF NOT EXISTS idx_q_images_question ON question_images(question_id)`)

	// The ON DELETE CASCADE is declarative only: this connection is opened
	// without foreign_keys=ON, so quiz.Delete removes these rows explicitly.
	Exec(QuizDB, `
		CREATE TABLE IF NOT EXISTS question_review_state (
			question_id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			repetitions INTEGER NOT NULL DEFAULT 0,
			interval_seconds INTEGER NOT NULL DEFAULT 0,
			ease_factor REAL NOT NULL DEFAULT 2.5,
			due_at DATETIME NOT NULL,
			last_rating TEXT NOT NULL,
			last_reviewed_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
		)
	`)
}

func CloseQuizDB() {
	if QuizDB != nil {
		QuizDB.Close()
	}
}

func ClosePromptDB() {
	if PromptDB != nil {
		PromptDB.Close()
	}
	CloseQuizDB()
}

func CloseBookmarkDB() {
	if BookmarkDB != nil {
		BookmarkDB.Close()
	}
}

func CloseHistoryDB() {
	if HistoryDB != nil {
		HistoryDB.Close()
	}
}

func CloseTodoDB() {
	if TodoDB != nil {
		TodoDB.Close()
	}
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
		_, err = TodoDB.Exec("INSERT INTO todos (user_id, title, description, domain, status) VALUES (?, ?, ?, ?, ?)",
			1, "Sample Todo", "This is a sample todo", "", "pending")
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
	InitPromptDB(dataPath)
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
	if PromptDB != nil {
		PromptDB.Close()
	}
}
