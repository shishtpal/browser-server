package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Sentinel errors returned by attachment operations. The chat service maps
// these to user-facing submit errors with distinct HTTP codes.
var (
	ErrAttachmentNotFound  = errors.New("attachment not found")
	ErrAttachmentNotStaged = errors.New("attachment is not staged")
	ErrAttachmentTooMany   = errors.New("too many attachments")
	ErrAttachmentTooLarge  = errors.New("attachment total bytes exceeded")
	// ErrAttachmentInvalidFilename is returned when a rename would persist an
	// empty or whitespace-only display filename.
	ErrAttachmentInvalidFilename = errors.New("invalid attachment filename")
)

const attachmentColumns = `id, conversation_id, COALESCE(message_id, ''), filename, content_type, size_bytes, width, height, storage_key, status, created_at`

// CreateStagedAttachment persists metadata for a freshly uploaded, validated
// image. The image bytes must already be on disk; storageKey is the
// server-generated relative filename that was used to store them. The caller
// supplies id (typically from NewID) so the DB primary key matches the
// filename prefix used by attachments.Write.
func (s *Store) CreateStagedAttachment(ctx context.Context, id, conversationID, filename, contentType string, sizeBytes int64, width, height int, storageKey string) (Attachment, error) {
	if err := s.requireConversation(ctx, conversationID); err != nil {
		return Attachment{}, err
	}
	now := time.Now().UTC()
	att := Attachment{
		ID:             id,
		ConversationID: conversationID,
		Filename:       filename,
		ContentType:    contentType,
		SizeBytes:      sizeBytes,
		Width:          width,
		Height:         height,
		StorageKey:     storageKey,
		Status:         "staged",
		CreatedAt:      now,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_message_attachments (id, conversation_id, filename, content_type, size_bytes, width, height, storage_key, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'staged', ?)`,
		att.ID, att.ConversationID, att.Filename, att.ContentType, att.SizeBytes, nullableInt(width), nullableInt(height), att.StorageKey, formatTime(now))
	if err != nil {
		return Attachment{}, err
	}
	return att, nil
}

// GetAttachment returns one attachment after verifying it belongs to the given
// conversation, so a client can never read another conversation's upload.
func (s *Store) GetAttachment(ctx context.Context, conversationID, attachmentID string) (Attachment, error) {
	var a Attachment
	row := s.db.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE id = ? AND conversation_id = ?`, attachmentID, conversationID)
	if err := scanAttachment(row, &a); err != nil {
		return Attachment{}, err
	}
	return a, nil
}

// RenameAttachment updates the display filename of an attachment after
// verifying it belongs to the given conversation. Returns the updated row.
// Filename is display-only; the on-disk StorageKey is never changed. An empty
// or whitespace-only filename is rejected before the ownership lookup so the
// store never persists a blank display name even if a caller skips the
// API-level sanitization.
func (s *Store) RenameAttachment(ctx context.Context, conversationID, attachmentID, filename string) (Attachment, error) {
	if strings.TrimSpace(filename) == "" {
		return Attachment{}, ErrAttachmentInvalidFilename
	}
	a, err := s.GetAttachment(ctx, conversationID, attachmentID)
	if err != nil {
		return Attachment{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE ai_message_attachments SET filename = ? WHERE id = ?`, filename, attachmentID); err != nil {
		return Attachment{}, err
	}
	a.Filename = filename
	return a, nil
}

// ListAttachmentsForMessage returns the attachments claimed by a message,
// ordered by upload time. Used to rebuild multimodal history for providers.
func (s *Store) ListAttachmentsForMessage(ctx context.Context, messageID string) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE message_id = ? ORDER BY created_at ASC, rowid ASC`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		if err := scanAttachment(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListStagedAttachments returns the still-unattached uploads for a conversation
// (used to restore the composer's pending previews after a reload).
func (s *Store) ListStagedAttachments(ctx context.Context, conversationID string) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE conversation_id = ? AND status = 'staged' ORDER BY created_at ASC, rowid ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attachment
	for rows.Next() {
		var a Attachment
		if err := scanAttachment(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAttachments returns committed (attached) image attachments across all
// conversations, newest first. Staged (unattached) uploads are excluded because
// they are ephemeral and scoped to a single conversation. limit is clamped to
// 1–500 (200 when unset or non-positive).
func (s *Store) ListAttachments(ctx context.Context, limit int) ([]Attachment, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE status = 'attached' ORDER BY created_at DESC, rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Attachment, 0)
	for rows.Next() {
		var a Attachment
		if err := scanAttachment(rows, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteStagedAttachment cancels a staged upload: it deletes the metadata row
// and returns the record (including its storage key) so the caller can remove
// the on-disk file. Attached attachments cannot be cancelled this way.
func (s *Store) DeleteStagedAttachment(ctx context.Context, conversationID, attachmentID string) (Attachment, error) {
	a, err := s.GetAttachment(ctx, conversationID, attachmentID)
	if err != nil {
		return Attachment{}, err
	}
	if a.Status != "staged" {
		return Attachment{}, ErrAttachmentNotStaged
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM ai_message_attachments WHERE id = ? AND status = 'staged'`, attachmentID); err != nil {
		return Attachment{}, err
	}
	return a, nil
}

// ExpireStagedAttachments removes staged uploads older than the cutoff and
// returns the removed records so the caller can delete their files. Bound by
// max so a single cleanup pass cannot stall the database.
//
// The DELETE filters by status='staged' to avoid removing a concurrently claimed
// attachment (race between cleanup SELECT and a submit BEGIN TURN). We track
// which rows were actually deleted so file removal is only attempted for those.
func (s *Store) ExpireStagedAttachments(ctx context.Context, olderThan time.Time, max int) ([]Attachment, error) {
	if max <= 0 || max > 1000 {
		max = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE status = 'staged' AND created_at < ? ORDER BY created_at ASC LIMIT ?`, formatTime(olderThan), max)
	if err != nil {
		return nil, err
	}
	var expired []Attachment
	for rows.Next() {
		var a Attachment
		if err := scanAttachment(rows, &a); err != nil {
			rows.Close()
			return nil, err
		}
		expired = append(expired, a)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(expired) == 0 {
		return nil, nil
	}
	// Build a parameterized IN clause to avoid SQL injection.
	placeholders := make([]string, len(expired))
	args := make([]any, len(expired))
	for i, a := range expired {
		placeholders[i] = "?"
		args[i] = a.ID
	}
	// Only delete rows still in 'staged' status; a concurrent claim may have
	// moved one to 'attached' between our SELECT and this DELETE.
	query := `DELETE FROM ai_message_attachments WHERE id IN (` + strings.Join(placeholders, ",") + `) AND status = 'staged'`
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == int64(len(expired)) {
		// All rows deleted — safe to remove every file.
		return expired, nil
	}
	// Partial delete: some rows were concurrently claimed. Only return the
	// ones that were actually removed. We re-query by ID to discover which
	// survived, keeping the query bounded.
	liveIDs := make(map[string]bool)
	liveRows, err := s.db.QueryContext(ctx, `SELECT id FROM ai_message_attachments WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		// Best-effort: if the re-query fails, return nothing to avoid
		// deleting files of claimed attachments.
		return nil, nil
	}
	defer liveRows.Close()
	for liveRows.Next() {
		var id string
		if err := liveRows.Scan(&id); err != nil {
			return nil, nil
		}
		liveIDs[id] = true
	}
	var removed []Attachment
	for _, a := range expired {
		if !liveIDs[a.ID] {
			removed = append(removed, a)
		}
	}
	return removed, nil
}

// CleanupExpiredAttachments is a convenience wrapper used by the module's
// startup and periodic cleanup job.
func (s *Store) CleanupExpiredAttachments(ctx context.Context, olderThan time.Time) ([]Attachment, error) {
	return s.ExpireStagedAttachments(ctx, olderThan, 500)
}

func (s *Store) requireConversation(ctx context.Context, conversationID string) error {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM conversations WHERE id = ?`, conversationID).Scan(&one)
	return err
}

func nullableInt(v int) any {
	if v <= 0 {
		return nil
	}
	return v
}

type attachmentScanner interface {
	Scan(dest ...any) error
}

func scanAttachment(row attachmentScanner, a *Attachment) error {
	var created string
	var width, height sql.NullInt64
	if err := row.Scan(&a.ID, &a.ConversationID, &a.MessageID, &a.Filename, &a.ContentType, &a.SizeBytes, &width, &height, &a.StorageKey, &a.Status, &created); err != nil {
		return err
	}
	a.CreatedAt = parseTime(created)
	if width.Valid {
		a.Width = int(width.Int64)
	}
	if height.Valid {
		a.Height = int(height.Int64)
	}
	return nil
}

// claimAttachmentsInTx atomically validates and claims staged attachment IDs
// inside an open transaction. It enforces the per-message image count and
// aggregate byte limits and returns the claimed attachments in upload order.
// Duplicate IDs in the list are rejected to prevent double-counting bytes.
func claimAttachmentsInTx(ctx context.Context, tx *sql.Tx, conversationID string, attachmentIDs []string, maxImages, maxTotalBytes int) ([]Attachment, error) {
	if len(attachmentIDs) == 0 {
		return nil, nil
	}
	if len(attachmentIDs) > maxImages {
		return nil, fmt.Errorf("%w: got %d, limit %d", ErrAttachmentTooMany, len(attachmentIDs), maxImages)
	}
	seen := make(map[string]bool, len(attachmentIDs))
	claimed := make([]Attachment, 0, len(attachmentIDs))
	var total int64
	for _, id := range attachmentIDs {
		if seen[id] {
			return nil, fmt.Errorf("%w: duplicate %s", ErrAttachmentNotFound, id)
		}
		seen[id] = true
		var a Attachment
		var created string
		var width, height sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT `+attachmentColumns+` FROM ai_message_attachments WHERE id = ? AND conversation_id = ?`, id, conversationID).
			Scan(&a.ID, &a.ConversationID, &a.MessageID, &a.Filename, &a.ContentType, &a.SizeBytes, &width, &height, &a.StorageKey, &a.Status, &created)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: %s", ErrAttachmentNotFound, id)
		}
		if err != nil {
			return nil, err
		}
		if a.Status != "staged" {
			return nil, fmt.Errorf("%w: %s", ErrAttachmentNotStaged, id)
		}
		a.CreatedAt = parseTime(created)
		if width.Valid {
			a.Width = int(width.Int64)
		}
		if height.Valid {
			a.Height = int(height.Int64)
		}
		total += a.SizeBytes
		if maxTotalBytes > 0 && total > int64(maxTotalBytes) {
			return nil, fmt.Errorf("%w: %d > %d", ErrAttachmentTooLarge, total, maxTotalBytes)
		}
		claimed = append(claimed, a)
	}
	return claimed, nil
}

func updateAttachmentClaimedTx(ctx context.Context, tx *sql.Tx, messageID string, claimed []Attachment) error {
	for _, a := range claimed {
		if _, err := tx.ExecContext(ctx, `UPDATE ai_message_attachments SET status = 'attached', message_id = ? WHERE id = ?`, messageID, a.ID); err != nil {
			return err
		}
	}
	return nil
}
