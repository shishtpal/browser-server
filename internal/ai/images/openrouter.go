package images

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"browser-server/internal/ai/attachments"
)

// openrouterRequest builds the OpenRouter image request body. Editing passes
// source images as data-URI input references.
func openrouterRequest(mc Model, r GenerateRequest, size string) (map[string]any, error) {
	payload := map[string]any{"model": mc.ID, "prompt": r.Prompt, "resolution": size, "n": 1}
	if r.AspectRatio != "" {
		payload["aspect_ratio"] = r.AspectRatio
	}
	if r.Seed != nil {
		payload["seed"] = *r.Seed
	}
	if len(r.Sources) > 0 {
		refs := make([]map[string]any, 0, len(r.Sources))
		for _, b := range r.Sources {
			contentType, _, _, err := attachments.ValidateImage(b, supportedImageTypes)
			if err != nil {
				return nil, err
			}
			refs = append(refs, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(b)}})
		}
		payload["input_references"] = refs
	}
	return payload, nil
}

func extractOpenRouter(b []byte) ([]byte, string, error) {
	var v struct {
		Data []struct {
			B64JSON   string `json:"b64_json"`
			MediaType string `json:"media_type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &v); err != nil || len(v.Data) == 0 || v.Data[0].B64JSON == "" {
		return nil, "", errors.New("image provider response contains no image")
	}
	out, err := base64.StdEncoding.DecodeString(v.Data[0].B64JSON)
	if err != nil {
		return nil, "", errors.New("invalid image data")
	}
	ct, _, _, err := attachments.ValidateImage(out, supportedImageTypes)
	if err != nil {
		return nil, "", err
	}
	if v.Data[0].MediaType != "" {
		ct = v.Data[0].MediaType
	}
	return out, ct, nil
}
