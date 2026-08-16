package tools

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-server/internal/ai/config"
	"browser-server/internal/ai/ocr"
	"browser-server/internal/ai/provider"
)

//go:embed schemas/ocr_image.json
var ocrImageSchema []byte

// visionCompleter is satisfied by provider.OpenAICompatibleClient; tests
// inject fakes so no outbound HTTP happens.
type visionCompleter interface {
	Complete(ctx context.Context, req provider.ChatRequest) (provider.ChatResponse, error)
}

// ocrClientFactory builds the vision client for one call. Default wraps
// provider.NewOpenAICompatibleClient (mirrors the memory synthesizer wiring).
type ocrClientFactory func(baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration) visionCompleter

// ocrPDFConvert runs pdftoppm; replaceable in tests.
type ocrPDFConvert func(ctx context.Context, cfg config.OCRPopplerConfig, pdfPath, outDir string, firstPage, lastPage int, additionalDirs []string) ([]string, error)

type ocrImageTool struct {
	cfg        config.OCRConfig
	cfgPath    string
	providers  map[string]config.ProviderConfig
	newClient  ocrClientFactory
	convertPDF ocrPDFConvert
	pdfDirs    []string // paths.additional_dirs, forwarded to Poppler resolution
	orSiteURL  string   // OpenRouter attribution site_url, forwarded to the client
	orAppName  string   // OpenRouter attribution app_name, forwarded to the client
}

// registerOCRImage wires the ocr_image tool. Providers come from the models
// file; cfgPath anchors relative ocr.output_dir values next to bs-ai-config.json.
// orCfg carries the editable OpenRouter attribution headers sent on vision calls.
func registerOCRImage(r *Registry, cfg config.OCRConfig, cfgPath string, providers map[string]config.ProviderConfig, additionalDirs []string, orCfg config.OpenRouterConfig) {
	t := &ocrImageTool{cfg: cfg, cfgPath: cfgPath, providers: providers, orSiteURL: orCfg.SiteURL, orAppName: orCfg.AppName}
	t.newClient = func(baseURL, apiKey string, timeout time.Duration, retryAttempts int, retryDelay time.Duration) visionCompleter {
		return provider.NewOpenAICompatibleClient(baseURL, apiKey, timeout, retryAttempts, retryDelay, t.orSiteURL, t.orAppName)
	}
	t.convertPDF = func(ctx context.Context, pc config.OCRPopplerConfig, pdfPath, outDir string, fp, lp int, extraDirs []string) ([]string, error) {
		return ocr.Convert(ctx, pc, pdfPath, outDir, fp, lp, extraDirs)
	}
	t.pdfDirs = additionalDirs
	r.add(Tool{
		Name:     "ocr_image",
		Category: "Vision",
		Description: "OCR a local image or PDF. " +
			"image_to_text (default) OCRs ONE image (png/jpg/webp/gif/tiff/bmp) via the configured vision provider and writes the extracted text to a .txt file (next to the image, or under ocr.output_dir when set); only a compact status JSON is returned — read the .txt with read_file. " +
			"pdf_to_images rasterizes a PDF into page images (report.pdf -> report-pdf-images/) using the Poppler CLI and returns the page list; then call image_to_text once per page — those calls may be made in parallel. PDFs are NEVER OCR'd directly: passing a PDF to image_to_text returns a hint instead. " +
			"Image bytes leave the machine to the configured vision provider.",
		Schema:  json.RawMessage(ocrImageSchema),
		Execute: t.execute,
	})
}

type ocrArgs struct {
	Action          string `json:"action"`
	Path            string `json:"path"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	OutputDir       string `json:"output_dir"`
	DPI             int    `json:"dpi"`
	FirstPage       int    `json:"first_page"`
	LastPage        int    `json:"last_page"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

var ocrArgFields = map[string]bool{
	"action": true, "path": true, "provider": true, "model": true,
	"prompt": true, "output_dir": true, "dpi": true,
	"first_page": true, "last_page": true, "max_output_tokens": true,
}

func (t *ocrImageTool) execute(ctx context.Context, raw json.RawMessage) (any, error) {
	var a ocrArgs
	if err := strict(raw, &a, ocrArgFields); err != nil {
		return nil, err
	}
	if strings.TrimSpace(a.Path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	switch a.Action {
	case "", "image_to_text":
		return t.imageToText(ctx, a)
	case "pdf_to_images":
		return t.pdfToImages(ctx, a)
	default:
		return nil, fmt.Errorf("unknown action %q (expected image_to_text or pdf_to_images)", a.Action)
	}
}

// pdfToImages rasterizes page ranges of a local PDF with Poppler and returns
// the list of generated page-image paths together with the follow-up hint.
func (t *ocrImageTool) pdfToImages(ctx context.Context, a ocrArgs) (any, error) {
	if a.Provider != "" || a.Model != "" || a.Prompt != "" || a.MaxOutputTokens != 0 {
		return nil, fmt.Errorf("provider/model/prompt/max_output_tokens are image_to_text arguments; pdf_to_images only accepts path, output_dir, dpi, first_page, last_page")
	}
	pdfPath := filepath.Clean(a.Path)
	info, err := os.Stat(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, expected a PDF file")
	}
	if !strings.EqualFold(filepath.Ext(pdfPath), ".pdf") {
		return nil, fmt.Errorf("pdf_to_images requires a .pdf file, got %q", pdfPath)
	}

	stem := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))
	outDir := a.OutputDir
	if outDir == "" {
		if t.cfg.OutputDir != "" {
			outDir = filepath.Join(t.resolveConfigPath(t.cfg.OutputDir), stem+"-pdf-images")
		} else {
			outDir = filepath.Join(filepath.Dir(pdfPath), stem+"-pdf-images")
		}
	} else if !filepath.IsAbs(outDir) {
		outDir = t.resolveConfigPath(outDir)
	}

	dpi := a.DPI
	if dpi == 0 {
		dpi = t.cfg.Poppler.DPI
	}
	pc := t.cfg.Poppler
	pc.DPI = dpi

	pages, err := t.convertPDF(ctx, pc, pdfPath, outDir, a.FirstPage, a.LastPage, t.pdfDirs)
	if err != nil {
		return nil, err
	}
	absPages := make([]string, len(pages))
	for i, p := range pages {
		absPages[i] = filepath.Join(outDir, filepath.Base(p))
	}

	return map[string]any{
		"ok":         true,
		"output_dir": outDir,
		"page_count": len(absPages),
		"pages":      absPages,
		"next_step":  "call image_to_text once per entry in pages; those calls may be made in parallel, then read each .txt with read_file",
	}, nil
}

var ocrImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true,
	".gif": true, ".tiff": true, ".tif": true, ".bmp": true,
}

// imageToText OCRs one image via a vision-capable OpenAI-compatible provider
// and atomically writes the extracted text to a .txt file on disk.
func (t *ocrImageTool) imageToText(ctx context.Context, a ocrArgs) (any, error) {
	imgPath := filepath.Clean(a.Path)
	if strings.EqualFold(filepath.Ext(imgPath), ".pdf") {
		// Steer small models towards the two-step flow instead of hallucinating.
		stem := strings.TrimSuffix(filepath.Base(imgPath), ".pdf")
		return map[string]any{
			"ok":               false,
			"converted":        false,
			"hint":             "PDF requires two steps",
			"suggested_action": "pdf_to_images",
			"pdf_images_dir":   filepath.Join(filepath.Dir(imgPath), stem+"-pdf-images"),
		}, nil
	}
	ext := strings.ToLower(filepath.Ext(imgPath))
	if !ocrImageExts[ext] {
		return nil, fmt.Errorf("unsupported image extension %q (supported: png, jpg, jpeg, webp, gif, tiff, tif, bmp)", ext)
	}
	info, err := os.Stat(imgPath)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, expected an image file")
	}
	maxBytes := t.cfg.MaxInputBytes
	if maxBytes <= 0 {
		maxBytes = 20 * 1024 * 1024
	}
	if info.Size() > int64(maxBytes) {
		return nil, fmt.Errorf("image is %d bytes, over ocr.max_input_bytes %d", info.Size(), maxBytes)
	}

	data, err := os.ReadFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	provName := a.Provider
	if provName == "" {
		provName = t.cfg.DefaultProvider
	}
	modelID := a.Model
	if modelID == "" {
		modelID = t.cfg.DefaultModel
	}
	pcfg, ok := t.providers[provName]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured in the models file", provName)
	}
	var model config.ModelConfig
	found := false
	for _, m := range pcfg.Models {
		if m.ID == modelID {
			model, found = m, true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("model %q not configured under provider %q", modelID, provName)
	}
	if !model.SupportsVision {
		return nil, fmt.Errorf("model %q has supports_vision=false in the models file; OCR requires a vision-capable model", modelID)
	}
	maxTokens := a.MaxOutputTokens
	if maxTokens == 0 {
		maxTokens = model.MaxOutputTokens
	}
	prompt := strings.TrimSpace(a.Prompt)
	if prompt == "" {
		prompt = "Extract all visible text from this image. Preserve order and layout where possible. Output only the extracted text, no commentary."
	}

	timeout := time.Duration(pcfg.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := t.newClient(pcfg.BaseURL, pcfg.APIKey, timeout, pcfg.RetryAttempts, time.Duration(pcfg.RetryDelaySeconds)*time.Second)

	started := time.Now()
	resp, err := client.Complete(ctx, provider.ChatRequest{
		Provider: provName,
		Model:    modelID,
		Messages: []provider.Message{{
			Role:    "user",
			Content: prompt,
			ImageParts: []provider.ImagePart{{
				DataURL: "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data),
			}},
		}},
		MaxOutputTokens: maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("vision call %s/%s: %w", provName, modelID, err)
	}
	text := strings.TrimSpace(resp.Content)
	if text == "" {
		return map[string]any{"ok": false, "reason": "empty_extraction"}, nil
	}

	outPath, err := t.outputTextPath(imgPath, a.OutputDir)
	if err != nil {
		return nil, err
	}
	tmp := outPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0644); err != nil {
		return nil, fmt.Errorf("write %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, outPath); err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("rename %q -> %q: %w", tmp, outPath, err)
	}

	return map[string]any{
		"ok":         true,
		"output":     outPath,
		"chars":      len(text),
		"provider":   provName,
		"model":      modelID,
		"latency_ms": time.Since(started).Milliseconds(),
	}, nil
}

// outputTextPath resolves where the OCR text file goes:
// explicit output_dir > shared ocr.output_dir (mirroring the image's relative
// parent) > next to the source image (idempotent overwrite).
func (t *ocrImageTool) outputTextPath(imgPath, outputDir string) (string, error) {
	stem := strings.TrimSuffix(filepath.Base(imgPath), filepath.Ext(imgPath))
	if outputDir != "" {
		if !filepath.IsAbs(outputDir) {
			outputDir = t.resolveConfigPath(outputDir)
		}
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return "", fmt.Errorf("create output dir %q: %w", outputDir, err)
		}
		return filepath.Join(outputDir, stem+".txt"), nil
	}
	if t.cfg.OutputDir != "" {
		root := t.resolveConfigPath(t.cfg.OutputDir)
		cfgDir := filepath.Dir(t.cfgPath)
		relParent, err := filepath.Rel(cfgDir, filepath.Dir(imgPath))
		if err != nil || strings.HasPrefix(relParent, "..") {
			root = filepath.Join(root, filepath.Base(filepath.Dir(imgPath)))
		} else if relParent != "." {
			root = filepath.Join(root, relParent)
		}
		if err := os.MkdirAll(root, 0755); err != nil {
			return "", fmt.Errorf("create output dir %q: %w", root, err)
		}
		return filepath.Join(root, stem+".txt"), nil
	}
	return filepath.Join(filepath.Dir(imgPath), stem+".txt"), nil
}

// resolveConfigPath resolves a possibly-relative config value against the
// bs-ai-config.json directory (same convention as ResolvePath).
func (t *ocrImageTool) resolveConfigPath(p string) string {
	if filepath.IsAbs(p) || t.cfgPath == "" {
		return p
	}
	return filepath.Join(filepath.Dir(t.cfgPath), p)
}
