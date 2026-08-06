package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func (s *Store) CreateConversation(ctx context.Context, title, provider, model, profile string) (Conversation, error) {
	now := time.Now().UTC()
	title = strings.TrimSpace(title)
	if title == "" {
		title = "New chat"
	}
	if len(title) > 120 {
		title = title[:120]
	}
	conversation := Conversation{
		ID:        NewID("conv"),
		Title:     title,
		Provider:  provider,
		Model:     model,
		Profile:   profile,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO conversations (id, title, provider, model, profile, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		conversation.ID, conversation.Title, conversation.Provider, conversation.Model, conversation.Profile, formatTime(now), formatTime(now))
	return conversation, err
}

// ForkConversation creates a new conversation seeded with a copy of the source
// conversation's messages, from the first message through uptoMessageID (inclusive).
// The new conversation inherits the source provider/model/profile/skills. Superseded
// messages and non-settled (pending) rows are skipped so only committed context is
// carried over. Returns sql.ErrNoRows if uptoMessageID is not part of the source.
func (s *Store) ForkConversation(ctx context.Context, sourceID, uptoMessageID string) (Conversation, error) {
	source, messages, err := s.GetConversation(ctx, sourceID)
	if err != nil {
		return Conversation{}, err
	}

	// Determine the cut-off index (inclusive). uptoMessageID must exist in the source.
	cutoff := -1
	for i, m := range messages {
		if m.ID == uptoMessageID {
			cutoff = i
			break
		}
	}
	if cutoff < 0 {
		return Conversation{}, sql.ErrNoRows
	}

	now := time.Now().UTC()
	title := strings.TrimSpace(source.Title)
	if title == "" {
		title = "New chat"
	}
	title = title + " (branch)"
	if len(title) > 120 {
		title = title[:120]
	}
	skillsJSON, _ := json.Marshal(source.Skills)

	forked := Conversation{
		ID:        NewID("conv"),
		Title:     title,
		Provider:  source.Provider,
		Model:     source.Model,
		Profile:   source.Profile,
		Skills:    source.Skills,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Conversation{}, err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx,
		`INSERT INTO conversations (id, title, provider, model, profile, skills, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		forked.ID, forked.Title, forked.Provider, forked.Model, forked.Profile, string(skillsJSON), formatTime(now), formatTime(now)); err != nil {
		return Conversation{}, err
	}

	// Preserve relative ordering with a monotonically increasing timestamp so the
	// copied messages sort identically to the source.
	base := now
	for i := 0; i <= cutoff; i++ {
		src := messages[i]
		if src.Status == "superseded" || src.Status == "pending" {
			continue
		}
		created := base.Add(time.Duration(i) * time.Millisecond)
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO messages (id, conversation_id, role, content, tool_call_id, status, created_at, reasoning) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			NewID("msg"), forked.ID, src.Role, src.Content, nullString(src.ToolCallID), src.Status, formatTime(created), src.Reasoning); err != nil {
			return Conversation{}, err
		}
	}

	if err = tx.Commit(); err != nil {
		return Conversation{}, err
	}
	return forked, nil
}

func (s *Store) ListConversations(ctx context.Context, query string, limit int) ([]Conversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	includeArchived, _ := ctx.Value("include_archived").(bool)
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(query) != "" {
		pattern := "%" + strings.TrimSpace(query) + "%"
		if includeArchived {
			rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.skills, c.archived, c.created_at, c.updated_at,
				COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
				FROM conversations c WHERE c.title LIKE ? ORDER BY c.updated_at DESC LIMIT ?`, pattern, limit)
		} else {
			rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.skills, c.archived, c.created_at, c.updated_at,
				COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
				FROM conversations c WHERE c.title LIKE ? AND c.archived = FALSE ORDER BY c.updated_at DESC LIMIT ?`, pattern, limit)
		}
	} else {
		if includeArchived {
			rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.skills, c.archived, c.created_at, c.updated_at,
				COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
				FROM conversations c ORDER BY c.updated_at DESC LIMIT ?`, limit)
		} else {
			rows, err = s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.skills, c.archived, c.created_at, c.updated_at,
				COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
				FROM conversations c WHERE c.archived = FALSE ORDER BY c.updated_at DESC LIMIT ?`, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conversations := make([]Conversation, 0)
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, item)
	}
	return conversations, rows.Err()
}

func (s *Store) GetConversation(ctx context.Context, id string) (Conversation, []Message, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, title, provider, model, profile, skills, archived, created_at, updated_at, '' FROM conversations WHERE id = ?`, id)
	conversation, err := scanConversation(row)
	if err != nil {
		return Conversation{}, nil, err
	}
	messages, err := s.ListMessages(ctx, id, 0)
	return conversation, messages, err
}

func (s *Store) UpdateConversation(ctx context.Context, id, title, provider, model string) (Conversation, error) {
	current, _, err := s.GetConversation(ctx, id)
	if err != nil {
		return Conversation{}, err
	}
	if strings.TrimSpace(title) != "" {
		current.Title = strings.TrimSpace(title)
		if len(current.Title) > 120 {
			current.Title = current.Title[:120]
		}
	}
	if provider != "" {
		current.Provider = provider
	}
	if model != "" {
		current.Model = model
	}
	current.UpdatedAt = time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `UPDATE conversations SET title = ?, provider = ?, model = ?, updated_at = ? WHERE id = ?`,
		current.Title, current.Provider, current.Model, formatTime(current.UpdatedAt), id)
	return current, err
}

// UpdateConversationSkills persists the active skills for a conversation.
func (s *Store) UpdateConversationSkills(ctx context.Context, id string, skills []string) error {
	data, _ := json.Marshal(skills)
	_, err := s.db.ExecContext(ctx, `UPDATE conversations SET skills = ?, updated_at = ? WHERE id = ?`,
		string(data), formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) ArchiveConversation(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE conversations SET archived = TRUE, updated_at = ? WHERE id = ?`,
		formatTime(time.Now().UTC()), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) RestoreConversation(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE conversations SET archived = FALSE, updated_at = ? WHERE id = ?`,
		formatTime(time.Now().UTC()), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListArchivedConversations(ctx context.Context, limit int) ([]Conversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, c.title, c.provider, c.model, c.profile, c.skills, c.archived, c.created_at, c.updated_at,
		COALESCE((SELECT content FROM messages WHERE conversation_id = c.id ORDER BY created_at DESC LIMIT 1), '') AS preview
		FROM conversations c WHERE c.archived = TRUE ORDER BY c.updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	conversations := make([]Conversation, 0)
	for rows.Next() {
		item, err := scanConversation(rows)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, item)
	}
	return conversations, rows.Err()
}

func (s *Store) DeleteConversation(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type conversationScanner interface {
	Scan(dest ...any) error
}

func scanConversation(row conversationScanner) (Conversation, error) {
	var item Conversation
	var created, updated, skillsJSON string
	if err := row.Scan(&item.ID, &item.Title, &item.Provider, &item.Model, &item.Profile, &skillsJSON, &item.Archived, &created, &updated, &item.Preview); err != nil {
		return Conversation{}, err
	}
	item.CreatedAt = parseTime(created)
	item.UpdatedAt = parseTime(updated)
	if len(item.Preview) > 160 {
		item.Preview = item.Preview[:160]
	}
	// Parse skills JSON array
	if skillsJSON != "" && skillsJSON != "[]" {
		_ = json.Unmarshal([]byte(skillsJSON), &item.Skills)
	}
	return item, nil
}
