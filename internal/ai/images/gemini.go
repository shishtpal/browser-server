package images

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"browser-server/internal/ai/attachments"
)

// geminiRequest builds the Gemini Interactions request body (text-to-image and
// image editing via inline image blocks).
func geminiRequest(mc Model, r GenerateRequest, size string) (map[string]any, error) {
	input := make([]any, 0, len(r.Sources)+1)
	for _, b := range r.Sources {
		ct, _, _, e := attachments.ValidateImage(b, supportedImageTypes)
		if e != nil {
			return nil, e
		}
		input = append(input, map[string]string{"type": "image", "mime_type": ct, "data": base64.StdEncoding.EncodeToString(b)})
	}
	input = append(input, map[string]string{"type": "text", "text": r.Prompt})
	payload := map[string]any{
		"model":           strings.TrimPrefix(mc.ID, "models/"),
		"input":           input,
		"response_format": map[string]string{"type": "image", "image_size": size},
	}
	if mc.ThinkingLevel != "" {
		payload["generation_config"] = map[string]any{"thinking_level": mc.ThinkingLevel}
	}
	return payload, nil
}

// block is a candidate image payload discovered while scanning a provider
// response.
type block struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MIMEType string `json:"mime_type"`
}

// extract walks the entire Interactions response for the generated image
// block. Gemini returns images inside steps[].content[] (type "image"), and
// the SDK-only output_image convenience field may or may not be present in the
// raw REST body. A recursive scan finds the image wherever it lives and prefers
// the block whose MIME type is most specific.
func extract(b []byte) ([]byte, string, error) {
	var root any
	if json.Unmarshal(b, &root) != nil {
		return nil, "", errors.New("malformed image provider response")
	}
	var best *block
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if blk, ok := asImageBlock(t); ok {
				// Prefer a block that carries an explicit image MIME type.
				if best == nil || (blk.MIMEType != "" && best.MIMEType == "") {
					best = blk
				}
			}
			for _, val := range t {
				walk(val)
			}
		case []any:
			for _, el := range t {
				walk(el)
			}
		}
	}
	walk(root)
	// Last-resort fallback: some model revisions return the image bytes under
	// unrecognized field names. Scan every base64-looking string and accept
	// the one that decodes to a real image (verified by magic bytes).
	if best == nil {
		if blk := findImageByMagic(root); blk != nil {
			best = blk
		}
	}
	if best == nil {
		return nil, "", fmt.Errorf("image provider response contains no image: %s", snippet(b))
	}
	out, e := base64.StdEncoding.DecodeString(stripDataURL(best.Data))
	if e != nil {
		return nil, "", errors.New("invalid image data")
	}
	ct, _, _, e := attachments.ValidateImage(out, supportedImageTypes)
	if e != nil {
		return nil, "", e
	}
	if best.MIMEType != "" {
		ct = best.MIMEType
	}
	return out, ct, nil
}

// asImageBlock recognizes an image payload object. Gemini image blocks may
// appear in several shapes depending on the model and SDK revision:
//   - {type:"image", mime_type:"image/png", data:"BASE64"}
//   - {type:"image", mimeType:"image/png", data:"BASE64"}        (camelCase)
//   - {inline_data:{mime_type,data}} / {inlineData:{mimeType,data}}
//   - the SDK output_image / outputImage convenience object
//
// We accept any block that carries inline image bytes and is either tagged as
// an image (type "image") or declares an image/* MIME type.
func asImageBlock(m map[string]any) (*block, bool) {
	data := firstString(m, "data", "bytes", "image_data", "imageData")
	mime := firstString(m, "mime_type", "mimeType")
	typ, _ := m["type"].(string)
	// Unwrap nested inline data carriers.
	for _, k := range []string{"inline_data", "inlineData"} {
		if inner, ok := m[k].(map[string]any); ok {
			if d := firstString(inner, "data", "bytes"); d != "" {
				data = d
			}
			if mt := firstString(inner, "mime_type", "mimeType"); mt != "" && mime == "" {
				mime = mt
			}
		}
	}
	if data == "" {
		return nil, false
	}
	if typ == "image" || strings.HasPrefix(mime, "image/") {
		return &block{Type: typ, Data: data, MIMEType: mime}, true
	}
	return nil, false
}

// findImageByMagic scans every string in the response for a base64 blob that
// decodes to a real image (PNG/JPEG/WEBP magic bytes) and returns the largest
// match. It is the fallback for responses whose image field names are unknown.
func findImageByMagic(v any) *block {
	var hit *block
	var scan func(v any)
	scan = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			for _, val := range t {
				scan(val)
			}
		case []any:
			for _, el := range t {
				scan(el)
			}
		case string:
			raw := stripDataURL(t)
			if len(raw) < 64 {
				return
			}
			dec, err := base64.StdEncoding.DecodeString(raw)
			if err != nil {
				return
			}
			if ct, _, _, err := attachments.ValidateImage(dec, supportedImageTypes); err == nil {
				if hit == nil || len(dec) > hit.size() {
					hit = &block{Data: raw, MIMEType: ct}
				}
			}
		}
	}
	scan(v)
	return hit
}

// size returns the decoded byte length for ordering candidates.
func (b *block) size() int {
	if b == nil {
		return 0
	}
	if d, err := base64.StdEncoding.DecodeString(stripDataURL(b.Data)); err == nil {
		return len(d)
	}
	return 0
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// stripDataURL removes an optional "data:image/...;base64," prefix and any
// embedded whitespace/newlines so the remaining string is a clean base64 blob.
func stripDataURL(s string) string {
	if i := strings.Index(s, ","); i > 0 {
		if strings.HasPrefix(s, "data:") {
			s = s[i+1:]
		}
	}
	return strings.NewReplacer("\n", "", "\r", "", "\t", "", " ", "").Replace(s)
}

// snippet returns a short, safely-truncated preview of a provider response so
// "no image" failures can be diagnosed without logging the full body.
func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 240 {
		s = s[:240]
	}
	return s
}
