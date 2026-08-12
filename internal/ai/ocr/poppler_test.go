package ocr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"browser-server/internal/ai/config"
)

func testPopplerConfig(dir string) config.OCRPopplerConfig {
	return config.OCRPopplerConfig{
		Dir:            dir,
		DPI:            150,
		Format:         "png",
		MaxPages:       5,
		TimeoutSeconds: 30,
	}
}

func TestResolveFromConfiguredDir(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, pdftoppmName())
	if err := os.WriteFile(exe, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(testPopplerConfig(dir), nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != exe {
		t.Fatalf("Resolve = %q, want %q", got, exe)
	}
}

func TestResolveMissing(t *testing.T) {
	if _, err := exec.LookPath(pdftoppmName()); err == nil {
		t.Skipf("skipping: pdftoppm exists on PATH (%s)", pdftoppmName())
	}
	_, err := Resolve(testPopplerConfig(""), []string{t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "poppler not found") {
		t.Fatalf("Resolve error = %v", err)
	}
}

func TestConvertPageCapEnforced(t *testing.T) {
	_, err := Convert(context.Background(), testPopplerConfig(""), "in.pdf", t.TempDir(), 1, 20, nil)
	if err == nil || !strings.Contains(err.Error(), "max_pages") {
		t.Fatalf("Convert error = %v, want max_pages mention", err)
	}
}

func TestConvertPageRangeInverted(t *testing.T) {
	_, err := Convert(context.Background(), testPopplerConfig(""), "in.pdf", t.TempDir(), 5, 2, nil)
	if err == nil || !strings.Contains(err.Error(), "before first_page") {
		t.Fatalf("Convert error = %v", err)
	}
}

func TestConvertHappyPathWithFakeRunner(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "pages")
	// Fake pdftoppm binary presence via Dir.
	exe := filepath.Join(dir, pdftoppmName())
	if err := os.WriteFile(exe, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}

	orig := runCommand
	defer func() { runCommand = orig }()
	var gotArgs []string
	runCommand = func(ctx context.Context, exeArg string, args ...string) error {
		if exeArg != exe {
			t.Errorf("exe = %q, want %q", exeArg, exe)
		}
		gotArgs = append([]string(nil), args...)
		for _, name := range []string{"page-01.png", "page-02.png", "page-03.png"} {
			if err := os.WriteFile(filepath.Join(outDir, name), []byte("img"), 0644); err != nil {
				return err
			}
		}
		return nil
	}

	pages, err := Convert(context.Background(), testPopplerConfig(dir), "in.pdf", outDir, 1, 3, nil)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("pages len = %d (%v)", len(pages), pages)
	}
	for i, want := range []string{"page-01.png", "page-02.png", "page-03.png"} {
		if filepath.Base(pages[i]) != want {
			t.Errorf("pages[%d] = %q, want %q", i, pages[i], want)
		}
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"-r 150", "-png", "-f 1", "-l 3", "in.pdf"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
}

func TestConvertNoPagesProduced(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()
	exe := filepath.Join(dir, pdftoppmName())
	os.WriteFile(exe, []byte("x"), 0755)

	orig := runCommand
	defer func() { runCommand = orig }()
	runCommand = func(ctx context.Context, exeArg string, args ...string) error { return nil }

	_, err := Convert(context.Background(), testPopplerConfig(dir), "in.pdf", outDir, 0, 0, nil)
	if err == nil || !strings.Contains(err.Error(), "produced no pages") {
		t.Fatalf("Convert error = %v", err)
	}
}

func TestConvertPropagatesRunnerError(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()
	exe := filepath.Join(dir, pdftoppmName())
	os.WriteFile(exe, []byte("x"), 0755)

	orig := runCommand
	defer func() { runCommand = orig }()
	runCommand = func(ctx context.Context, exeArg string, args ...string) error {
		return errors.New("boom")
	}

	_, err := Convert(context.Background(), testPopplerConfig(dir), "in.pdf", outDir, 1, 2, nil)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Convert error = %v", err)
	}
}
