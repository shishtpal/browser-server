package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// newTestStore opens a fresh store and seeds one conversation, returning the
// store and conversation id.
func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	c, err := s.CreateConversation(context.Background(), "test", "p", "m", "")
	if err != nil {
		t.Fatal(err)
	}
	return s, c.ID
}

func staged(t *testing.T, s *Store, convID, id string) Attachment {
	t.Helper()
	att, err := s.CreateStagedAttachment(context.Background(), id, convID, "f.png", "image/png", 1024, 8, 8, id+".png")
	if err != nil {
		t.Fatalf("CreateStagedAttachment: %v", err)
	}
	return att
}

func TestCreateStagedAttachmentRejectsUnknownConversation(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.CreateStagedAttachment(context.Background(), "att-1", "missing-conv", "f.png", "image/png", 10, 1, 1, "att-1.png")
	if err == nil {
		t.Fatal("expected error for unknown conversation")
	}
}

func TestGetAttachmentIsScopedToConversation(t *testing.T) {
	s, convA := newTestStore(t)
	convB, err := s.CreateConversation(context.Background(), "b", "p", "m", "")
	if err != nil {
		t.Fatal(err)
	}
	staged(t, s, convA, "att-A")
	// Cross-conversation retrieval must fail — a client can never read another
	// conversation's upload even with the right attachment id.
	if _, err := s.GetAttachment(context.Background(), convB.ID, "att-A"); err == nil {
		t.Fatal("expected not found for cross-conversation get")
	}
	// Same conversation retrieval succeeds.
	if _, err := s.GetAttachment(context.Background(), convA, "att-A"); err != nil {
		t.Fatalf("expected ok for same conversation, got %v", err)
	}
}

func TestDeleteStagedAttachmentOnlyCancelsStaged(t *testing.T) {
	s, convID := newTestStore(t)
	att := staged(t, s, convID, "att-1")
	if _, err := s.DeleteStagedAttachment(context.Background(), convID, att.ID); err != nil {
		t.Fatalf("DeleteStagedAttachment: %v", err)
	}
	if _, err := s.GetAttachment(context.Background(), convID, att.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
	// An already-attached upload cannot be cancelled.
	att2 := staged(t, s, convID, "att-2")
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{att2.ID}, 5, 1<<20); err != nil {
		t.Fatalf("BeginTurnWithAttachments: %v", err)
	}
	if _, err := s.DeleteStagedAttachment(context.Background(), convID, att2.ID); !errors.Is(err, ErrAttachmentNotStaged) {
		t.Fatalf("expected ErrAttachmentNotStaged for attached upload, got %v", err)
	}
}

func TestBeginTurnWithAttachmentsRejectsDuplicateIDs(t *testing.T) {
	s, convID := newTestStore(t)
	staged(t, s, convID, "att-1")
	// Passing the same id twice must reject to prevent double-counting bytes.
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{"att-1", "att-1"}, 5, 1<<20); err == nil {
		t.Fatal("expected error for duplicate attachment ids")
	}
}

func TestBeginTurnWithAttachmentsRejectsUnknownAndCrossConversation(t *testing.T) {
	s, convID := newTestStore(t)
	other, err := s.CreateConversation(context.Background(), "other", "p", "m", "")
	if err != nil {
		t.Fatal(err)
	}
	staged(t, s, other.ID, "att-other")
	// Unknown id -> not found.
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{"missing"}, 5, 1<<20); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("expected ErrAttachmentNotFound for unknown id, got %v", err)
	}
	// Id from a different conversation -> not found (ownership enforced).
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{"att-other"}, 5, 1<<20); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("expected ErrAttachmentNotFound for cross-conversation id, got %v", err)
	}
}

func TestBeginTurnWithAttachmentsEnforcesCountAndBytes(t *testing.T) {
	s, convID := newTestStore(t)
	staged(t, s, convID, "att-1")
	staged(t, s, convID, "att-2")
	// maxImages=1 but two ids requested -> too many.
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{"att-1", "att-2"}, 1, 1<<20); !errors.Is(err, ErrAttachmentTooMany) {
		t.Fatalf("expected ErrAttachmentTooMany, got %v", err)
	}
	// maxTotalBytes below the staged size -> too large.
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{"att-1"}, 5, 512); !errors.Is(err, ErrAttachmentTooLarge) {
		t.Fatalf("expected ErrAttachmentTooLarge, got %v", err)
	}
}

func TestBeginTurnWithAttachmentsClaimsAndHydratesOnReload(t *testing.T) {
	s, convID := newTestStore(t)
	staged(t, s, convID, "att-1")
	staged(t, s, convID, "att-2")

	user, _, claimed, err := s.BeginTurnWithAttachments(context.Background(), convID, "describe these", []string{"att-1", "att-2"}, 5, 1<<20)
	if err != nil {
		t.Fatalf("BeginTurnWithAttachments: %v", err)
	}
	if len(claimed) != 2 || claimed[0].ID != "att-1" {
		t.Fatalf("claimed = %+v", claimed)
	}
	// Both attachments are now attached and bound to the user message id.
	for _, a := range claimed {
		got, err := s.GetAttachment(context.Background(), convID, a.ID)
		if err != nil {
			t.Fatalf("GetAttachment: %v", err)
		}
		if got.Status != "attached" || got.MessageID != user.ID {
			t.Fatalf("attachment not claimed: %+v", got)
		}
	}
	// Reloading the conversation hydrates attachments onto the user message.
	msgs, err := s.ListMessages(context.Background(), convID, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	var found bool
	for _, m := range msgs {
		if m.ID == user.ID {
			found = true
			if len(m.Attachments) != 2 {
				t.Fatalf("user message attachments not hydrated: %+v", m.Attachments)
			}
		}
	}
	if !found {
		t.Fatalf("user message %s not found in reload", user.ID)
	}
}

func TestExpireStagedAttachmentsRemovesOnlyOldRows(t *testing.T) {
	s, convID := newTestStore(t)
	old := staged(t, s, convID, "att-old")
	// Manually age the old attachment past the cutoff.
	if _, err := s.db.Exec(`UPDATE ai_message_attachments SET created_at = ? WHERE id = ?`, formatTime(time.Now().UTC().Add(-2*time.Hour)), old.ID); err != nil {
		t.Fatal(err)
	}
	staged(t, s, convID, "att-fresh")

	removed, err := s.ExpireStagedAttachments(context.Background(), time.Now().UTC().Add(-time.Hour), 100)
	if err != nil {
		t.Fatalf("ExpireStagedAttachments: %v", err)
	}
	if len(removed) != 1 || removed[0].ID != "att-old" {
		t.Fatalf("removed = %+v, want [att-old]", removed)
	}
	// The fresh staged attachment survives.
	if _, err := s.GetAttachment(context.Background(), convID, "att-fresh"); err != nil {
		t.Fatalf("fresh attachment should still exist: %v", err)
	}
	// The expired one is gone.
	if _, err := s.GetAttachment(context.Background(), convID, "att-old"); err == nil {
		t.Fatal("expected old attachment removed")
	}
}

func TestListAttachmentsForConversationReturnsAll(t *testing.T) {
	s, convID := newTestStore(t)
	staged(t, s, convID, "att-1")
	staged(t, s, convID, "att-2")
	atts, err := s.ListAttachmentsForConversation(context.Background(), convID)
	if err != nil {
		t.Fatalf("ListAttachmentsForConversation: %v", err)
	}
	if len(atts) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(atts))
	}
}

func TestListAttachmentsReturnsOnlyAttachedAcrossConversations(t *testing.T) {
	s, convA := newTestStore(t)
	convB, err := s.CreateConversation(context.Background(), "b", "p", "m", "")
	if err != nil {
		t.Fatal(err)
	}
	// Staged-only uploads must be excluded from the cross-conversation gallery.
	staged(t, s, convA, "att-staged")
	// Claim one attachment in each conversation so both become "attached".
	staged(t, s, convA, "att-A")
	staged(t, s, convB.ID, "att-B")
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convA, "hi", []string{"att-A"}, 5, 1<<20); err != nil {
		t.Fatalf("BeginTurnWithAttachments convA: %v", err)
	}
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convB.ID, "hi", []string{"att-B"}, 5, 1<<20); err != nil {
		t.Fatalf("BeginTurnWithAttachments convB: %v", err)
	}

	atts, err := s.ListAttachments(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListAttachments: %v", err)
	}
	// Exactly the two attached uploads, never the staged-only one.
	if len(atts) != 2 {
		t.Fatalf("expected 2 attached attachments, got %d: %+v", len(atts), atts)
	}
	ids := map[string]bool{}
	for _, a := range atts {
		ids[a.ID] = true
		if a.Status != "attached" {
			t.Fatalf("attachment %s is not attached: %s", a.ID, a.Status)
		}
		if a.ConversationID != convA && a.ConversationID != convB.ID {
			t.Fatalf("attachment %s has unexpected conversation %s", a.ID, a.ConversationID)
		}
	}
	if !ids["att-A"] || !ids["att-B"] {
		t.Fatalf("expected att-A and att-B, got %v", ids)
	}
	if ids["att-staged"] {
		t.Fatal("staged-only attachment must not appear in gallery list")
	}
}

func TestListAttachmentsClampsLimit(t *testing.T) {
	s, convID := newTestStore(t)
	// Attach three uploads so the limit can be exercised.
	for _, id := range []string{"a1", "a2", "a3"} {
		staged(t, s, convID, id)
		if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{id}, 5, 1<<20); err != nil {
			t.Fatalf("BeginTurnWithAttachments %s: %v", id, err)
		}
	}
	// limit <= 0 defaults to 200, returning all three here.
	all, err := s.ListAttachments(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListAttachments default: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 with default limit, got %d", len(all))
	}
	// limit=2 caps the result, newest first.
	capped, err := s.ListAttachments(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListAttachments limit=2: %v", err)
	}
	if len(capped) != 2 {
		t.Fatalf("expected 2 with limit=2, got %d", len(capped))
	}
	if capped[0].ID != "a3" {
		t.Fatalf("expected newest (a3) first, got %s", capped[0].ID)
	}
	// limit over 500 is clamped down to 500 (still returns all available).
	over, err := s.ListAttachments(context.Background(), 9999)
	if err != nil {
		t.Fatalf("ListAttachments limit=9999: %v", err)
	}
	if len(over) != 3 {
		t.Fatalf("expected 3 after clamp, got %d", len(over))
	}
}

func TestRenameAttachmentUpdatesFilename(t *testing.T) {
	s, convID := newTestStore(t)
	att := staged(t, s, convID, "att-1")
	// Claim the upload so the row is "attached" — rename must work on attached
	// attachments (the gallery only lists attached ones).
	if _, _, _, err := s.BeginTurnWithAttachments(context.Background(), convID, "hi", []string{att.ID}, 5, 1<<20); err != nil {
		t.Fatalf("BeginTurnWithAttachments: %v", err)
	}

	renamed, err := s.RenameAttachment(context.Background(), convID, att.ID, "new-name.png")
	if err != nil {
		t.Fatalf("RenameAttachment: %v", err)
	}
	if renamed.ID != att.ID || renamed.Filename != "new-name.png" {
		t.Fatalf("renamed = %+v, want id %s filename new-name.png", renamed, att.ID)
	}
	// The rename persists, and never touches the on-disk storage key or status.
	got, err := s.GetAttachment(context.Background(), convID, att.ID)
	if err != nil {
		t.Fatalf("GetAttachment after rename: %v", err)
	}
	if got.Filename != "new-name.png" {
		t.Fatalf("stored filename = %q, want new-name.png", got.Filename)
	}
	if got.StorageKey != att.StorageKey || got.Status != "attached" {
		t.Fatalf("rename must not alter storage/status: storage=%q status=%q", got.StorageKey, got.Status)
	}
}

func TestRenameAttachmentNotFound(t *testing.T) {
	s, convID := newTestStore(t)
	// Unknown id in an existing conversation -> not found.
	if _, err := s.RenameAttachment(context.Background(), convID, "att-missing", "x.png"); !IsNotFound(err) {
		t.Fatalf("expected not found for unknown attachment, got %v", err)
	}
	// Another conversation's attachment -> not found (ownership enforced).
	other, err := s.CreateConversation(context.Background(), "other", "p", "m", "")
	if err != nil {
		t.Fatal(err)
	}
	staged(t, s, convID, "att-A")
	if _, err := s.RenameAttachment(context.Background(), other.ID, "att-A", "x.png"); !IsNotFound(err) {
		t.Fatalf("expected not found for cross-conversation rename, got %v", err)
	}
	// The rejected cross-conversation attempt must not rename the original.
	if got, err := s.GetAttachment(context.Background(), convID, "att-A"); err != nil || got.Filename != "f.png" {
		t.Fatalf("cross-conversation attempt must not rename: got=%+v err=%v", got, err)
	}
}

func TestRenameAttachmentRejectsEmptyFilename(t *testing.T) {
	s, convID := newTestStore(t)
	staged(t, s, convID, "att-1")
	for _, name := range []string{"", "   ", "\t\n"} {
		if _, err := s.RenameAttachment(context.Background(), convID, "att-1", name); !errors.Is(err, ErrAttachmentInvalidFilename) {
			t.Fatalf("expected ErrAttachmentInvalidFilename for %q, got %v", name, err)
		}
	}
	// Rejecting must leave the stored filename unchanged.
	got, err := s.GetAttachment(context.Background(), convID, "att-1")
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if got.Filename != "f.png" {
		t.Fatalf("filename changed after rejected renames: %q", got.Filename)
	}
}
