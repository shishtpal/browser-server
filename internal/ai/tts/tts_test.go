package tts

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func testService(t *testing.T, handler http.HandlerFunc, voices []Voice) *Service {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := Config{
		Enabled:         true,
		DefaultProvider: "openrouter",
		Providers: map[string]Provider{
			"openrouter": {
				Type:                  providerTypeOpenRouter,
				BaseURL:               srv.URL,
				APIKey:                "test-key",
				RequestTimeoutSeconds: 5,
				Models: []Model{{
					ID:      "deepgram/flux-tts:free",
					Label:   "Flux",
					Default: true,
					Voices:  voices,
				}},
			},
		},
	}
	svc, err := New(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func TestGenerateHappyPath(t *testing.T) {
	const audio = "ID3fake-mp3-bytes"
	var gotAuth, gotModel, gotInput, gotVoice string
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Errorf("path = %s, want /audio/speech", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("HTTP-Referer") != refererURL {
			t.Errorf("HTTP-Referer = %q", r.Header.Get("HTTP-Referer"))
		}
		if r.Header.Get("Referer") != refererURL {
			t.Errorf("Referer = %q", r.Header.Get("Referer"))
		}
		if r.Header.Get("X-OpenRouter-Title") != openRouterTitle {
			t.Errorf("X-OpenRouter-Title = %q", r.Header.Get("X-OpenRouter-Title"))
		}
		body, _ := io.ReadAll(r.Body)
		gotAuth = r.Header.Get("Authorization")
		payload := string(body)
		if !strings.Contains(payload, `"model":"deepgram/flux-tts:free"`) {
			t.Errorf("payload missing model: %s", payload)
		}
		if !strings.Contains(payload, `"input":"Hello world"`) {
			t.Errorf("payload missing input: %s", payload)
		}
		if !strings.Contains(payload, `"voice":"flux-gemma-en"`) {
			t.Errorf("payload missing default voice: %s", payload)
		}
		gotModel, gotInput, gotVoice = "ok", "ok", "ok"
		w.Header().Set("X-Generation-Id", "gen_abc")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(audio))
	}, []Voice{{ID: "flux-gemma-en", Label: "Gemma (EN)"}})

	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hello world"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if gotAuth == "" || gotModel == "" || gotInput == "" || gotVoice == "" {
		t.Fatal("provider was not called with the expected payload")
	}
	if !strings.HasPrefix(x.ID, "tts_") {
		t.Fatalf("id = %q, want tts_ prefix", x.ID)
	}
	if x.Filename != x.ID+".mp3" || x.ContentType != contentTypeMPEG {
		t.Fatalf("file meta = %+v", x)
	}
	if x.GenerationID != "gen_abc" {
		t.Fatalf("generation_id = %q, want gen_abc", x.GenerationID)
	}
	if x.SizeBytes != int64(len(audio)) || x.Provider != "openrouter" || x.Voice != "flux-gemma-en" {
		t.Fatalf("speech = %+v", x)
	}
	data, err := os.ReadFile(svc.FilePath(x.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != audio {
		t.Fatalf("persisted bytes = %q, want %q", data, audio)
	}
	got, err := svc.Get(context.Background(), x.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello world" || got.GenerationID != "gen_abc" {
		t.Fatalf("Get = %+v", got)
	}
	listed, err := svc.List(context.Background(), 10)
	if err != nil || len(listed) != 1 || listed[0].ID != x.ID {
		t.Fatalf("List = %+v, %v", listed, err)
	}
}

func TestGenerateProviderErrorsLeaveNoFile(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{"too many requests", http.StatusTooManyRequests},
		{"server error", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":{"message":"upstream failed"}}`))
			}, nil)
			_, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hello"})
			if !errors.Is(err, ErrProvider) {
				t.Fatalf("err = %v, want ErrProvider", err)
			}
			entries, _ := os.ReadDir(svc.root)
			if len(entries) != 0 {
				t.Fatalf("gallery should be empty after provider error, got %v", entries)
			}
			listed, err := svc.List(context.Background(), 10)
			if err != nil || len(listed) != 0 {
				t.Fatalf("List = %+v, %v", listed, err)
			}
		})
	}
}

func TestGenerateErrorBodyOverSizeLimitReportsStatus(t *testing.T) {
	// An oversized non-2xx body must surface the provider status/error, not
	// the audio-size cap message (regression: status is checked before size).
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", maxAudioBytes+2)))
	}, nil)
	_, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi"})
	if err == nil || !errors.Is(err, ErrProvider) {
		t.Fatalf("err = %v, want ErrProvider", err)
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("err = %v, want status 500 surfaced over size cap", err)
	}
}

func TestGenerateRejectsUnknownProviderAndModel(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("provider should not be called")
	}, []Voice{{ID: "flux-gemma-en", Label: "Gemma"}})

	if _, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi", Provider: "missing"}); err == nil || !strings.Contains(err.Error(), "unknown tts provider") {
		t.Fatalf("unknown provider: %v", err)
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi", Model: "nope"}); err == nil || !strings.Contains(err.Error(), "unknown tts model") {
		t.Fatalf("unknown model: %v", err)
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi", Voice: "nope"}); err == nil || !strings.Contains(err.Error(), "unknown voice") {
		t.Fatalf("unknown voice: %v", err)
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{}); err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("empty text: %v", err)
	}
	long := strings.Repeat("a", maxTextChars+1)
	if utf8.RuneCountInString(long) <= maxTextChars {
		t.Fatal("test fixture is not long enough")
	}
	if _, err := svc.Generate(context.Background(), GenerateRequest{Text: long}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize text: %v", err)
	}
}

func TestGenerateOmitsVoiceForVoicelessModel(t *testing.T) {
	// Models without configured voices (e.g. fish-audio) must send no
	// "voice" key at all; an empty string triggers a provider 400
	// ("too_small") instead of the provider defaulting the voice.
	var payload string
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3"))
	}, nil)
	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, `"voice"`) {
		t.Fatalf("payload should omit voice key: %s", payload)
	}
	if x.Voice != "" {
		t.Fatalf("voice = %q, want empty", x.Voice)
	}
}

func TestGenerateRejectsVoiceForVoicelessModel(t *testing.T) {
	// A model with no configured voices cannot honor an explicit Voice:
	// we have no id to validate against and the provider would use its
	// own default, so Generate must fail loudly instead of silently
	// sending an unexpected key or downgrading.
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3"))
	}, nil)
	_, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi", Voice: "custom-voice"})
	if err == nil || !strings.Contains(err.Error(), "no configured voices") {
		t.Fatalf("Generate error = %v, want unconfigurable-voice rejection", err)
	}
}

// fishService spins up a fake fish.audio endpoint so Fish Audio provider
// tests don't share the OpenRouter-shaped testService helper.
func fishService(t *testing.T, handler http.HandlerFunc) *Service {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := Config{
		Enabled:         true,
		DefaultProvider: "fish.audio",
		Providers: map[string]Provider{
			"fish.audio": {
				Type:                  providerTypeFishAudio,
				BaseURL:               srv.URL,
				APIKey:                "fish-key",
				RequestTimeoutSeconds: 5,
				Models: []Model{
					{ID: "s2.1-pro", Label: "Pro"},
					{ID: "s2.1-pro-free", Label: "Pro (free)", Default: true, ResponseFormat: "mp3"},
				},
			},
		},
	}
	svc, err := New(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

func TestGenerateFishAudioSendsModelInHeaderAndFishFields(t *testing.T) {
	// Fish Audio's contract differs from OpenRouter: model travels in the
	// `model` header, the body uses `text`/`reference_id`/`format`, and the
	// endpoint is /v1/tts (here served at /tts because the test server URL
	// already includes /v1).
	var gotMethod, gotPath, gotAuth, gotModel, gotContentType string
	var payload string
	svc := fishService(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotModel = r.Header.Get("model")
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		payload = string(body)
		w.Header().Set("X-Generation-Id", "gen_fish")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ID3fish-mp3"))
	})

	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hello fish"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/tts" {
		t.Fatalf("request = %s %s, want POST /tts", gotMethod, gotPath)
	}
	if gotAuth != "Bearer fish-key" {
		t.Fatalf("Authorization = %q, want Bearer fish-key", gotAuth)
	}
	if gotModel != "s2.1-pro-free" {
		t.Fatalf("model header = %q, want s2.1-pro-free", gotModel)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if strings.Contains(payload, `"model"`) {
		t.Fatalf("payload must not include model key when model travels in header: %s", payload)
	}
	if !strings.Contains(payload, `"text":"Hello fish"`) {
		t.Fatalf("payload missing text: %s", payload)
	}
	if strings.Contains(payload, `"reference_id"`) {
		t.Fatalf("payload should omit reference_id when no voice configured: %s", payload)
	}
	if !strings.Contains(payload, `"format":"mp3"`) {
		t.Fatalf("payload missing format from model.response_format: %s", payload)
	}
	if x.Provider != "fish.audio" || x.Model != "s2.1-pro-free" || x.Voice != "" {
		t.Fatalf("speech = %+v", x)
	}
	if x.GenerationID != "gen_fish" {
		t.Fatalf("generation_id = %q, want gen_fish", x.GenerationID)
	}
}

func TestGenerateFishAudioAcceptsVoiceAsReferenceID(t *testing.T) {
	// When the caller passes a voice id, Fish Audio expects it as
	// `reference_id` in the body (not `voice`).
	var payload string
	svc := fishService(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3"))
	})
	// Register a voice on the default model so the unknown-voice check passes.
	for i := range svc.cfg.Providers["fish.audio"].Models {
		if svc.cfg.Providers["fish.audio"].Models[i].Default {
			svc.cfg.Providers["fish.audio"].Models[i].Voices = []Voice{{ID: "ann", Label: "Ann"}}
		}
	}
	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi", Voice: "ann"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(payload, `"reference_id":"ann"`) {
		t.Fatalf("payload missing reference_id:ann: %s", payload)
	}
	if strings.Contains(payload, `"voice"`) {
		t.Fatalf("payload must use reference_id, not voice: %s", payload)
	}
	if x.Voice != "ann" {
		t.Fatalf("voice = %q, want ann", x.Voice)
	}
}

func TestLoadAcceptsFishAudioProvider(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, modelsFile, `{
  "enabled": true,
  "default_provider": "fish.audio",
  "providers": {
    "fish.audio": {
      "type": "fish_audio",
      "api_key": "env:TTS_FISH_KEY",
      "models": [
        {"id": "s2.1-pro", "label": "Pro"},
        {"id": "s2.1-pro-free", "label": "Pro free", "default": true, "response_format": "mp3"}
      ]
    }
  }
}`)
	t.Setenv("TTS_FISH_KEY", "fish-secret")
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p := cfg.Providers["fish.audio"]
	if p.APIKey != "fish-secret" {
		t.Fatalf("api_key = %q, want resolved secret", p.APIKey)
	}
	if p.BaseURL != defaultFishBaseURL {
		t.Fatalf("base_url = %q, want %q", p.BaseURL, defaultFishBaseURL)
	}
}

func TestLoadStillRejectsUnknownType(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, modelsFile, `{
  "enabled": true,
  "default_provider": "x",
  "providers": {
    "x": {"type": "nope", "api_key": "k", "models": [{"id": "m", "label": "M"}]}
  }
}`)
	if _, err := Load(dir); err == nil {
		t.Fatal("unknown provider type was accepted")
	}
}

func TestDeleteRemovesFileAndRow(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3"))
	}, nil)
	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), x.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(svc.FilePath(x.Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still present: %v", err)
	}
	if _, err := svc.Get(context.Background(), x.ID); err == nil {
		t.Fatal("row still present")
	}
}

func TestReadReturnsBytes(t *testing.T) {
	svc := testService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mp3-data"))
	}, nil)
	x, err := svc.Generate(context.Background(), GenerateRequest{Text: "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	got, data, err := svc.Read(context.Background(), x.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != x.ID || string(data) != "mp3-data" {
		t.Fatalf("Read = %+v %q", got, data)
	}
	if filepath.Base(svc.FilePath(x.Filename)) != x.Filename {
		t.Fatal("FilePath should stay under the gallery root")
	}
}
