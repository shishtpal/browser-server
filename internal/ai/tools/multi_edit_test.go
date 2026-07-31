package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiEditAppliesSequentialEditsAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	first := writeMultiEditFixture(t, dir, "first.txt", "alpha\nbeta\n")
	second := writeMultiEditFixture(t, dir, "second.txt", "one one\n")

	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: first, Find: "alpha", Replace: "ALPHA"},
		{Path: first, Find: "ALPHA\nbeta", Replace: "done"},
		{Path: second, Find: "one", Replace: "two", All: true},
	}, false))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["files_changed"] != 2 || response["total_replacements"] != 4 {
		t.Fatalf("unexpected response: %#v", response)
	}
	assertMultiEditContent(t, first, "done\n")
	assertMultiEditContent(t, second, "two two\n")
}

func TestMultiEditFailureDoesNotModifyAnyFile(t *testing.T) {
	dir := t.TempDir()
	first := writeMultiEditFixture(t, dir, "first.txt", "before\n")
	second := writeMultiEditFixture(t, dir, "second.txt", "current\n")

	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: first, Find: "before", Replace: "after"},
		{Path: second, Find: "missing", Replace: "replacement"},
	}, false))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["ok"] != false || response["error"] != "no_match" || response["op"] != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if response["nearest"] == nil {
		t.Fatalf("expected nearest-line hint: %#v", response)
	}
	assertMultiEditContent(t, first, "before\n")
	assertMultiEditContent(t, second, "current\n")
}

func TestMultiEditRejectsAmbiguousMatchUnlessAll(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "same same\n")
	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: path, Find: "same", Replace: "changed"},
	}, false))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["error"] != "ambiguous_match" || response["found"] != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
	assertMultiEditContent(t, path, "same same\n")
}

func TestMultiEditDryRunReturnsDiffWithoutWriting(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "before\n")
	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: path, Find: "before", Replace: "after"},
	}, true))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	diff, _ := response["diff"].(string)
	if response["dry_run"] != true || !strings.Contains(diff, "-before\n+after\n") {
		t.Fatalf("unexpected response: %#v", response)
	}
	assertMultiEditContent(t, path, "before\n")
}

func TestMultiEditDryRunShowsTrailingNewlineRemoval(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "only line\n")
	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: path, Find: "\n", Replace: ""},
	}, true))
	if err != nil {
		t.Fatal(err)
	}
	diff := result.(map[string]any)["diff"].(string)
	if !strings.Contains(diff, "+only line\n\\ No newline at end of file\n") {
		t.Fatalf("diff does not show trailing newline removal: %s", diff)
	}
	assertMultiEditContent(t, path, "only line\n")
}

func TestMultiEditDryRunMarksUnterminatedContextAndEmptyRange(t *testing.T) {
	t.Run("unterminated context", func(t *testing.T) {
		path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "one\ntwo")
		result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
			{Path: path, Find: "one", Replace: "ONE"},
		}, true))
		if err != nil {
			t.Fatal(err)
		}
		diff := result.(map[string]any)["diff"].(string)
		if !strings.Contains(diff, " two\n\\ No newline at end of file\n") {
			t.Fatalf("diff does not mark unterminated context: %s", diff)
		}
	})

	t.Run("whole file deletion", func(t *testing.T) {
		path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "one")
		result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
			{Path: path, Find: "one", Replace: ""},
		}, true))
		if err != nil {
			t.Fatal(err)
		}
		diff := result.(map[string]any)["diff"].(string)
		if !strings.Contains(diff, "@@ -1,1 +0,0 @@") {
			t.Fatalf("diff has invalid empty range: %s", diff)
		}
	})
}

func TestMultiEditDryRunBothSidesNoTrailingNewlineMiddleChange(t *testing.T) {
	// Both old and new lack a trailing newline, with changes in the middle.
	// Verifies the diff correctly marks "No newline at end of file" for both sides.
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "first\nsecond\nthird")
	result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: path, Find: "second", Replace: "CHANGED"},
	}, true))
	if err != nil {
		t.Fatal(err)
	}
	diff := result.(map[string]any)["diff"].(string)
	if !strings.Contains(diff, "-second") || !strings.Contains(diff, "+CHANGED") {
		t.Fatalf("diff missing change lines: %s", diff)
	}
	if !strings.Contains(diff, "\\ No newline at end of file") {
		t.Fatalf("diff does not mark missing trailing newline: %s", diff)
	}
	assertMultiEditContent(t, path, "first\nsecond\nthird")
}

func TestMultiEditPreservesCRLFAndTrailingNewline(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "before\r\nafter")
	if _, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
		{Path: path, Find: "before\nafter", Replace: "changed\nafter"},
	}, false)); err != nil {
		t.Fatal(err)
	}
	assertMultiEditContent(t, path, "changed\r\nafter")
}

func TestMultiEditRejectsNonTextAndRelativePaths(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "binary.dat")
	if err := os.WriteFile(binary, []byte{'a', 0, 'b'}, 0644); err != nil {
		t.Fatal(err)
	}
	tests := []multiEditOp{
		{Path: "relative.txt", Find: "a", Replace: "b"},
		{Path: binary, Find: "a", Replace: "b"},
	}
	for _, edit := range tests {
		if _, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{edit}, false)); err == nil {
			t.Fatalf("expected edit rejection for %q", edit.Path)
		}
	}
}

func TestMultiEditStrictlyValidatesEachEdit(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "before\n")
	for _, args := range []string{
		`{"edits":[{"path":` + quoted(path) + `,"find":"before","replace":"after","unknown":true}]}`,
		`{"edits":[{"path":` + quoted(path) + `,"find":"before"}]}`,
	} {
		if _, err := multiEdit(context.Background(), json.RawMessage(args)); err == nil {
			t.Fatalf("expected invalid edit rejection for %s", args)
		}
	}
}

func TestMultiEditRejectsNullBooleans(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "before\n")
	for _, args := range []string{
		`{"edits":[{"path":` + quoted(path) + `,"find":"before","replace":"after"}],"dry_run":null}`,
		`{"edits":[{"path":` + quoted(path) + `,"find":"before","replace":"after","all":null}]}`,
	} {
		if _, err := multiEdit(context.Background(), json.RawMessage(args)); err == nil {
			t.Fatalf("expected null boolean rejection for %s", args)
		}
	}
	assertMultiEditContent(t, path, "before\n")
}

func TestMultiEditBoundsResultAndDryRunOutput(t *testing.T) {
	t.Run("result size", func(t *testing.T) {
		path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "a")
		if _, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
			{Path: path, Find: "a", Replace: strings.Repeat("b", maxEditFileSize+1)},
		}, false)); err == nil {
			t.Fatal("expected result size rejection")
		}
		assertMultiEditContent(t, path, "a")
	})

	t.Run("dry-run output", func(t *testing.T) {
		path := writeMultiEditFixture(t, t.TempDir(), "test.txt", strings.Repeat("a", defaultMaxOutput))
		result, err := multiEdit(context.Background(), multiEditArgs(t, []multiEditOp{
			{Path: path, Find: "a", Replace: "b", All: true},
		}, true))
		if err != nil {
			t.Fatal(err)
		}
		response := result.(map[string]any)
		if response["error"] != "diff_too_large" {
			t.Fatalf("unexpected response: %#v", response)
		}
		assertMultiEditContent(t, path, strings.Repeat("a", defaultMaxOutput))
	})
}

func TestMultiEditDetectsConflictBeforeCommit(t *testing.T) {
	path := writeMultiEditFixture(t, t.TempDir(), "test.txt", "original\n")
	snapshot, err := loadForMultiEdit(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.text = "edited\n"
	if err := os.WriteFile(path, []byte("external\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := commitMultiEditFiles(context.Background(), []*multiEditSnapshot{snapshot}); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("expected conflict, got %v", err)
	}
	assertMultiEditContent(t, path, "external\n")
}

func TestMultiEditRollsBackWithStagedBackup(t *testing.T) {
	dir := t.TempDir()
	firstPath := writeMultiEditFixture(t, dir, "first.txt", "first\n")
	secondPath := writeMultiEditFixture(t, dir, "second.txt", "second\n")
	first, err := loadForMultiEdit(firstPath, 0)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadForMultiEdit(secondPath, 1)
	if err != nil {
		t.Fatal(err)
	}
	first.text, second.text = "FIRST\n", "SECOND\n"
	renames := 0
	rename := func(oldPath, newPath string) error {
		renames++
		if renames == 2 {
			return os.ErrPermission
		}
		return os.Rename(oldPath, newPath)
	}
	if err := commitMultiEditFilesWithRename(context.Background(), []*multiEditSnapshot{first, second}, rename); err == nil {
		t.Fatal("expected injected commit failure")
	}
	assertMultiEditContent(t, firstPath, "first\n")
	assertMultiEditContent(t, secondPath, "second\n")
}

func TestMultiEditSchemaIsValidJSON(t *testing.T) {
	schema := New().tools["multi_edit"].Schema
	if !json.Valid(schema) {
		t.Fatalf("invalid multi_edit schema: %s", schema)
	}
}

func writeMultiEditFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func multiEditArgs(t *testing.T, edits []multiEditOp, dryRun bool) json.RawMessage {
	t.Helper()
	args, err := json.Marshal(map[string]any{"edits": edits, "dry_run": dryRun})
	if err != nil {
		t.Fatal(err)
	}
	return args
}

func assertMultiEditContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("file content = %q, want %q", content, want)
	}
}
