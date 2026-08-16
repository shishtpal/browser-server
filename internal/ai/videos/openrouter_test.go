package videos

import (
	"net/http"
	"testing"
)

func TestOpenRouterCreatePayload(t *testing.T) {
	r := GenerateRequest{
		Prompt: "a slow push-in on a neon sign",
		Params: map[string]any{
			"duration":       "6",
			"size":           "1280x720",
			"seed":           42,
			"generate_audio": true,
			"frame_images": []any{
				"https://example.com/first.png",
				"https://example.com/last.png",
			},
			"input_references": []any{"https://example.com/ref.png"},
			"unused_param":     "dropped",
		},
	}
	payload, err := openrouterVideoProvider{}.createPayload(Model{ID: "google/veo-3.1-lite"}, r)
	if err != nil {
		t.Fatalf("createPayload: %v", err)
	}
	if got := payload["model"]; got != "google/veo-3.1-lite" {
		t.Fatalf("model = %v", got)
	}
	if got := payload["prompt"]; got != r.Prompt {
		t.Fatalf("prompt = %v", got)
	}
	if got := payload["duration"]; got != 6 {
		t.Fatalf("duration = %#v, want int 6", got)
	}
	if got := payload["seed"]; got != 42 {
		t.Fatalf("seed = %#v, want 42", got)
	}
	if got := payload["size"]; got != "1280x720" {
		t.Fatalf("size = %#v", got)
	}
	if got := payload["generate_audio"]; got != true {
		t.Fatalf("generate_audio = %#v", got)
	}
	if _, ok := payload["unused_param"]; ok {
		t.Fatal("unused_param should not be forwarded")
	}
	frames, ok := payload["frame_images"].([]any)
	if !ok || len(frames) != 2 {
		t.Fatalf("frame_images = %#v", payload["frame_images"])
	}
	wantTypes := []string{"first_frame", "last_frame"}
	for i, ref := range frames {
		m := ref.(map[string]any)
		if m["frame_type"] != wantTypes[i] {
			t.Errorf("frame %d type = %v, want %s", i, m["frame_type"], wantTypes[i])
		}
		iu := m["image_url"].(map[string]any)
		if iu["url"] == "" {
			t.Errorf("frame %d missing url", i)
		}
		if m["type"] != "image_url" {
			t.Errorf("frame %d type tag = %v", i, m["type"])
		}
	}
	refs, ok := payload["input_references"].([]any)
	if !ok || len(refs) != 1 {
		t.Fatalf("input_references = %#v", payload["input_references"])
	}
	if _, hasFrame := refs[0].(map[string]any)["frame_type"]; hasFrame {
		t.Error("input_references must not carry a frame_type")
	}
}

func TestOpenRouterCreatePayloadSingleFrame(t *testing.T) {
	r := GenerateRequest{Prompt: "x", Params: map[string]any{"frame_images": []any{"https://example.com/f.png"}}}
	payload, err := openrouterVideoProvider{}.createPayload(Model{ID: "m"}, r)
	if err != nil {
		t.Fatalf("createPayload: %v", err)
	}
	frames := payload["frame_images"].([]any)
	if got := frames[0].(map[string]any)["frame_type"]; got != "first_frame" {
		t.Fatalf("single frame type = %v, want first_frame", got)
	}
}

func TestOpenRouterCreatePayloadRejectsNonIntegerDuration(t *testing.T) {
	r := GenerateRequest{Prompt: "x", Params: map[string]any{"duration": "soon"}}
	if _, err := (openrouterVideoProvider{}).createPayload(Model{ID: "m"}, r); err == nil {
		t.Fatal("non-integer duration should be rejected")
	}
}

func TestValidateOpenRouterConstraints(t *testing.T) {
	tooMany := map[string]any{"frame_images": []any{"https://a", "https://b", "https://c"}}
	if err := validateOpenRouterConstraints(tooMany); err == nil {
		t.Fatal("three frame images should be rejected")
	}
	two := map[string]any{"frame_images": []any{"https://a", "https://b"}}
	if err := validateOpenRouterConstraints(two); err != nil {
		t.Fatalf("two frame images rejected: %v", err)
	}
	none := map[string]any{}
	if err := validateOpenRouterConstraints(none); err != nil {
		t.Fatalf("no frame images rejected: %v", err)
	}
}

func TestParseOpenRouterConfig(t *testing.T) {
	content := []byte(`{
		"enabled": true,
		"default_provider": "openrouter",
		"db_path": "ai-videos.db",
		"video_dir": "ai-videos",
		"providers": {
			"openrouter": {
				"type": "openrouter_video",
				"api_key": "test-key",
				"base_url": "https://openrouter.ai/",
				"models": [
					{
						"id": "google/veo-3.1-lite",
						"label": "Google Veo 3.1 Lite",
						"default": true,
						"parameters": [
							{"key": "prompt", "type": "text", "label": "Prompt", "required": true},
							{"key": "duration", "type": "select", "label": "Duration", "options": ["4", "6", "8"], "default": "6"},
							{"key": "size", "type": "select", "label": "Size", "options": ["1280x720", "720x1280"], "default": "1280x720"}
						]
					}
				]
			}
		}
	}`)
	cfg, err := parseConfig(content, "bs-ai-video-models.json")
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	p, ok := cfg.Providers["openrouter"]
	if !ok {
		t.Fatal("openrouter provider missing")
	}
	if p.BaseURL != "https://openrouter.ai" {
		t.Fatalf("base_url = %q, want origin", p.BaseURL)
	}
	if p.APIKey != "test-key" {
		t.Fatalf("api_key = %q", p.APIKey)
	}
	if len(p.Models) != 1 || p.Models[0].ID != "google/veo-3.1-lite" {
		t.Fatalf("models = %#v", p.Models)
	}
}

func TestParseOpenRouterConfigRejectsUnknownType(t *testing.T) {
	content := []byte(`{
		"enabled": true,
		"default_provider": "bad",
		"providers": {
			"bad": {
				"type": "spaceship_video",
				"api_key": "k",
				"models": [
					{"id": "m", "parameters": [{"key": "prompt", "type": "text", "label": "Prompt"}]}
				]
			}
		}
	}`)
	if _, err := parseConfig(content, "bs-ai-video-models.json"); err == nil {
		t.Fatal("unknown provider type should be rejected")
	}
}

func TestOpenRouterStatusMapping(t *testing.T) {
	cases := []struct {
		status string
		want   VideoStatus
	}{
		{"pending", StatusQueued},
		{"in_progress", StatusProgress},
		{"completed", StatusCompleted},
		{"failed", StatusFailed},
		{"cancelled", StatusFailed},
		{"expired", StatusFailed},
		{"mystery_future_state", StatusQueued},
	}
	for _, c := range cases {
		if got := statusFromOpenRouter(c.status); got != c.want {
			t.Errorf("%q → %v, want %v", c.status, got, c.want)
		}
	}
}

func TestSetOpenRouterHeadersAttribution(t *testing.T) {
	const (
		siteURL = "https://example.com/app"
		appName = "My App"
	)
	t.Run("openrouter target attaches attribution", func(t *testing.T) {
		h := http.Header{}
		setOpenRouterHeadersAttribution(h, Provider{BaseURL: "https://openrouter.ai", OpenRouterSiteURL: siteURL, OpenRouterAppName: appName})
		if h.Get("HTTP-Referer") != siteURL {
			t.Errorf("HTTP-Referer = %q, want %q", h.Get("HTTP-Referer"), siteURL)
		}
		if h.Get("Referer") != siteURL {
			t.Errorf("Referer = %q, want %q", h.Get("Referer"), siteURL)
		}
		if h.Get("X-Title") != appName {
			t.Errorf("X-Title = %q, want %q", h.Get("X-Title"), appName)
		}
	})
	t.Run("non-openrouter target gets nothing", func(t *testing.T) {
		h := http.Header{}
		setOpenRouterHeadersAttribution(h, Provider{BaseURL: "https://apihub.agnes-ai.com", OpenRouterSiteURL: siteURL, OpenRouterAppName: appName})
		if h.Get("HTTP-Referer") != "" || h.Get("Referer") != "" || h.Get("X-Title") != "" {
			t.Errorf("non-openrouter request got attribution headers: %v", h)
		}
	})
	t.Run("empty app name omits X-Title", func(t *testing.T) {
		h := http.Header{}
		setOpenRouterHeadersAttribution(h, Provider{BaseURL: "https://openrouter.ai", OpenRouterSiteURL: siteURL})
		if h.Get("HTTP-Referer") != siteURL {
			t.Errorf("HTTP-Referer = %q, want %q", h.Get("HTTP-Referer"), siteURL)
		}
		if h.Get("X-Title") != "" {
			t.Errorf("X-Title = %q, want empty", h.Get("X-Title"))
		}
	})
	t.Run("empty site url skips everything", func(t *testing.T) {
		h := http.Header{}
		setOpenRouterHeadersAttribution(h, Provider{BaseURL: "https://openrouter.ai", OpenRouterAppName: appName})
		if h.Get("HTTP-Referer") != "" || h.Get("Referer") != "" || h.Get("X-Title") != "" {
			t.Errorf("empty site_url still set headers: %v", h)
		}
	})
}
