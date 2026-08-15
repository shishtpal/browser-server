package images

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"browser-server/internal/ai/attachments"
)

// agnesRequest builds the Agnes Image generations request body. Editing passes
// source images as data-URI entries inside extra_body.image, and the output
// format is requested via extra_body.response_format (a top-level
// response_format is rejected by the API).
func agnesRequest(mc Model, r GenerateRequest, size string) (map[string]any, error) {
	extra := map[string]any{"response_format": "b64_json"}
	if len(r.Sources) > 0 {
		refs := make([]string, 0, len(r.Sources))
		for _, b := range r.Sources {
			contentType, _, _, err := attachments.ValidateImage(b, supportedImageTypes)
			if err != nil {
				return nil, err
			}
			refs = append(refs, "data:"+contentType+";base64,"+base64.StdEncoding.EncodeToString(b))
		}
		extra["image"] = refs
	}
	payload := map[string]any{
		"model":      mc.ID,
		"prompt":     r.Prompt,
		"size":       size,
		"extra_body": extra,
	}
	if r.AspectRatio != "" {
		payload["ratio"] = r.AspectRatio
	}
	return payload, nil
}

// extractAgnes reads the Agnes generations response. It prefers the Base64
// image, falling back to downloading the returned URL when the service only
// emits a link.
func (s *Service) extractAgnes(ctx context.Context, b []byte) ([]byte, string, error) {
	var v struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &v); err != nil || len(v.Data) == 0 {
		return nil, "", errors.New("image provider response contains no image")
	}
	if v.Data[0].B64JSON != "" {
		out, err := base64.StdEncoding.DecodeString(v.Data[0].B64JSON)
		if err != nil {
			return nil, "", errors.New("invalid image data")
		}
		ct, _, _, err := attachments.ValidateImage(out, supportedImageTypes)
		if err != nil {
			return nil, "", err
		}
		return out, ct, nil
	}
	if v.Data[0].URL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.Data[0].URL, nil)
		if err != nil {
			return nil, "", errors.New("invalid image url")
		}
		c := *s.client
		c.Timeout = 120 * time.Second
		resp, err := c.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %v", ErrProvider, err)
		}
		defer resp.Body.Close()
		out, err := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
		if err != nil {
			return nil, "", errors.New("failed to download generated image")
		}
		ct, _, _, err := attachments.ValidateImage(out, supportedImageTypes)
		if err != nil {
			return nil, "", err
		}
		return out, ct, nil
	}
	return nil, "", errors.New("image provider response contains no image")
}
