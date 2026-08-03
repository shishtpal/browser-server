package store

import "fmt"

func (s *Store) migrate() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
		`INSERT INTO schema_version (version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL CHECK (role IN ('system','user','assistant','tool')),
			content TEXT NOT NULL DEFAULT '',
			tool_call_id TEXT,
			status TEXT NOT NULL CHECK (status IN ('pending','completed','error','cancelled','superseded')),
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id TEXT PRIMARY KEY,
			conversation_id TEXT REFERENCES conversations(id) ON DELETE SET NULL,
			message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			request_payload TEXT,
			response_payload TEXT,
			payload_truncated INTEGER NOT NULL DEFAULT 0,
			http_status INTEGER,
			prompt_tokens INTEGER,
			completion_tokens INTEGER,
			total_tokens INTEGER,
			latency_ms INTEGER NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('success','error','cancelled')),
			error_code TEXT,
			error_message TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tool_calls (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL REFERENCES request_logs(id) ON DELETE CASCADE,
			message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
			tool_name TEXT NOT NULL,
			arguments TEXT NOT NULL,
			result TEXT,
			error_message TEXT,
			status TEXT NOT NULL CHECK (status IN ('success','error','cancelled','rejected')),
			duration_ms INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_conversation_created ON request_logs(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_calls_request ON tool_calls(request_id)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	// Incremental migrations keyed by schema_version
	migrations := []struct {
		version    int
		statements []string
	}{
		{2, []string{
			`ALTER TABLE conversations ADD COLUMN profile TEXT NOT NULL DEFAULT ''`,
		}},
		{3, []string{
			`ALTER TABLE conversations ADD COLUMN skills TEXT NOT NULL DEFAULT '[]'`,
		}},
		{4, []string{
			`ALTER TABLE conversations ADD COLUMN archived BOOLEAN DEFAULT FALSE`,
		}},
		{5, []string{
			`CREATE TABLE IF NOT EXISTS ai_message_attachments (
				id TEXT PRIMARY KEY,
				conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
				message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
				filename TEXT NOT NULL,
				content_type TEXT NOT NULL,
				size_bytes INTEGER NOT NULL,
				width INTEGER,
				height INTEGER,
				storage_key TEXT NOT NULL,
				status TEXT NOT NULL CHECK (status IN ('staged','attached')),
				created_at TEXT NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS idx_attachments_conversation ON ai_message_attachments(conversation_id)`,
			`CREATE INDEX IF NOT EXISTS idx_attachments_message ON ai_message_attachments(message_id)`,
			`CREATE INDEX IF NOT EXISTS idx_attachments_staged_created ON ai_message_attachments(status, created_at)`,
		}},
	}
	var currentVersion int
	s.db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&currentVersion)
	for _, m := range migrations {
		if currentVersion >= m.version {
			continue
		}
		for _, stmt := range m.statements {
			if _, err := s.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration v%d: %w", m.version, err)
			}
		}
		if _, err := s.db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, m.version); err != nil {
			return err
		}
	}

	return nil
}
