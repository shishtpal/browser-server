package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/provider"
)

type fakeVisionClient struct {
	resp provider.ChatResponse
	err  error
	req  provider.ChatRequest
}

func (f *fakeVisionClient) Complete(ctx context.Context, req provider.ChatRequest) (provider.ChatResponse, error) {
	f.req = req
	return f.resp, f.err
}

func newOCRTool(t *testing.T, client visionCompleter) *ocrImageTool {
	t.Helper()
	cfg := config.OCRConfig{
		Enabled:         true,
		DefaultProvider: "testprov",
		DefaultModel:    "vision-model",
		MaxInputBytes:   20 * 1024 * 1024,
		Poppler: config.OCRPopplerConfig{
			DPI: 150, Format: "png", MaxPages: 5, TimeoutSeconds: 30,
		},
	}
	tool := &ocrImageTool{
		cfg:     cfg,
		cfgPath: filepath.Join(t.TempDir(), "bs-ai-config.json"),
		providers: map[string]config.ProviderConfig{
			"testprov": {
				BaseURL: "https://example.test/v1",
				APIKey:  "sk-test",
				Models: []config.ModelConfig{
					{ID: "vision-model", SupportsVision: true, MaxOutputTokens: 1024},
					{ID: "text-model", SupportsVision: false, MaxOutputTokens: 512},
				},
			},
		},
		convertPDF: func(ctx context.Context, pc config.OCRPopplerConfig, pdfPath, outDir string, fp, lp int, extra []string) ([]string, error) {
			return nil, errors.New("not used in this test")
		},
	}
	tool.newClient = func(baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration) visionCompleter {
		return client
	}
	return tool
}

func writeTinyImage(t *testing.T, dir, name string) string {
	t.Helper()
	// 1x1 transparent PNG (68 bytes) — enough for stat/read; the vision call
	// is faked so the bytes never leave the machine.
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
		0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, png, 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOCRImageRequiresPath(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	_, err := tool.execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImageRejectsUnknownArg(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":"x.png","bogus":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown argument") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImagePDFHint(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	dir := t.TempDir()
	pdf := filepath.Join(dir, "report.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.4"), 0644)

	out, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(pdf)+`}`))
	if err != nil {
		t.Fatalf("expected hint, got error %v", err)
	}
	m := out.(map[string]any)
	if m["ok"] != false || m["suggested_action"] != "pdf_to_images" {
		t.Fatalf("hint payload = %v", m)
	}
	if !strings.Contains(m["pdf_images_dir"].(string), "report-pdf-images") {
		t.Fatalf("pdf_images_dir = %v", m["pdf_images_dir"])
	}
}

func TestOCRImageToTextWritesAtomicTxt(t *testing.T) {
	fc := &fakeVisionClient{resp: provider.ChatResponse{Content: "hello world"}}
	tool := newOCRTool(t, fc)
	dir := t.TempDir()
	img := writeTinyImage(t, dir, "scan.png")

	out, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m := out.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("result = %v", m)
	}
	txtPath := strings.TrimSuffix(img, ".png") + ".txt"
	if m["output"] != txtPath {
		t.Fatalf("output = %q, want %q", m["output"], txtPath)
	}
	data, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("txt not written: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("txt content = %q", data)
	}
	if _, err := os.Stat(txtPath + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file must be renamed away")
	}
	// Sanity-check the wire request went through the image part plumbing.
	msgs := fc.req.Messages
	if len(msgs) != 1 || len(msgs[0].ImageParts) != 1 || !strings.HasPrefix(msgs[0].ImageParts[0].DataURL, "data:image/png;base64,") {
		t.Fatalf("image parts = %+v", msgs)
	}
}

func TestOCRImageEmptyExtraction(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{resp: provider.ChatResponse{Content: "  "}})
	img := writeTinyImage(t, t.TempDir(), "blank.png")
	out, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m := out.(map[string]any)
	if m["ok"] != false || m["reason"] != "empty_extraction" {
		t.Fatalf("result = %v", m)
	}
}

func TestOCRImageNonVisionModelRejected(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	img := writeTinyImage(t, t.TempDir(), "photo.png")
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`,"model":"text-model"}`))
	if err == nil || !strings.Contains(err.Error(), "supports_vision") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImageUnknownProvider(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	img := writeTinyImage(t, t.TempDir(), "photo.png")
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`,"provider":"nope"}`))
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImageOversize(t *testing.T) {
	fc := &fakeVisionClient{}
	tool := newOCRTool(t, fc)
	// It's a 1x1 png, but even that is ~68 bytes; set limit to 10 to force.
	tool.cfg.MaxInputBytes = 10
	img := writeTinyImage(t, t.TempDir(), "big.png")
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`}`))
	if err == nil || !strings.Contains(err.Error(), "max_input_bytes") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImageUnsupportedExtension(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	f := filepath.Join(t.TempDir(), "x.txt")
	os.WriteFile(f, []byte("hi"), 0644)
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(f)+`}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported image extension") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRImageProviderErrorPropagates(t *testing.T) {
	perr := &provider.Error{Code: "rate_limited", Status: 429, Retryable: true, Diagnostic: "slow down"}
	tool := newOCRTool(t, &fakeVisionClient{err: perr})
	img := writeTinyImage(t, t.TempDir(), "scan.png")
	_, err := tool.execute(context.Background(), json.RawMessage(`{"path":`+jq(img)+`}`))
	if err == nil || !strings.Contains(err.Error(), "rate_limited") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRPDFToImagesRejectsVisionArgs(t *testing.T) {
	tool := newOCRTool(t, &fakeVisionClient{})
	dir := t.TempDir()
	pdf := filepath.Join(dir, "doc.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.4"), 0644)
	_, err := tool.execute(context.Background(), json.RawMessage(`{"action":"pdf_to_images","path":`+jq(pdf)+`,"model":"vision-model"}`))
	if err == nil || !strings.Contains(err.Error(), "image_to_text arguments") {
		t.Fatalf("err = %v", err)
	}
}

func TestOCRPDFToImagesDefaultOutDir(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "report.pdf")
	os.WriteFile(pdf, []byte("%PDF-1.4 fake"), 0644)

	fc := &fakeVisionClient{}
	tool := newOCRTool(t, fc)
	tool.convertPDF = func(ctx context.Context, pc config.OCRPopplerConfig, pdfPath, outDir string, fp, lp int, extra []string) ([]string, error) {
		names := []string{"page-01.png", "page-02.png"}
		for _, n := range names {
			os.WriteFile(filepath.Join(outDir, n), []byte("img"), 0644)
		}
		return []string{filepath.Join(outDir, names[0]), filepath.Join(outDir, names[1])}, nil
	}

	out, err := tool.execute(context.Background(), json.RawMessage(`{"action":"pdf_to_images","path":`+jq(pdf)+`}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m := out.(map[string]any)
	if m["ok"] != true {
		t.Fatalf("result = %v", m)
	}
	outDir := m["output_dir"].(string)
	if filepath.Base(outDir) != "report-pdf-images" {
		t.Fatalf("output_dir = %q", outDir)
	}
	pages := m["pages"].([]string)
	if len(pages) != 2 || !strings.HasSuffix(pages[0], "page-01.png") {
		t.Fatalf("pages = %v", pages)
	}
	hint := m["next_step"].(string)
	if !strings.Contains(hint, "parallel") {
		t.Fatalf("next_step = %q", hint)
	}
}

// jq JSON-quotes a Go string.
func jq(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
