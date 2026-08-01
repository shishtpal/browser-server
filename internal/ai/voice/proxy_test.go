package voice

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestProxyForwardsAudioFlushAndTranscript(t *testing.T) {
	received := make(chan map[string]any, 1)
	flush := make(chan bool, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Api-Subscription-Key") != "server-secret" {
			t.Error("upstream API key missing")
		}
		if r.URL.Query().Get("model") != "saaras:v4" || r.URL.Query().Get("language-code") != "en-IN" || r.URL.Query().Get("input_audio_codec") != "pcm_s16le" || r.URL.Query().Get("flush_signal") != "true" {
			t.Errorf("bad upstream query: %s", r.URL.RawQuery)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		var audio map[string]any
		if err := conn.ReadJSON(&audio); err != nil {
			return
		}
		received <- audio
		_, control, err := conn.ReadMessage()
		flush <- err == nil && string(control) == `{"type":"flush"}`
		_ = conn.WriteJSON(map[string]any{"type": "data", "data": map[string]string{"transcript": "hello"}})
	}))
	defer upstream.Close()

	upstreamURL := "ws" + strings.TrimPrefix(upstream.URL, "http")
	cfg := &Config{
		Enabled:   true,
		Languages: []Language{{Code: "en-IN", Label: "English"}},
		Recording: Recording{MaxFrameBytes: 1024, MaxAudioBytes: 4096},
		Providers: map[string]Provider{"sarvam": {
			Type: "sarvam_streaming", Enabled: true, BaseURL: upstreamURL, APIKey: "server-secret", RequestTimeoutSeconds: 5,
			Models: []Model{{ID: "saaras:v4", Label: "Saaras", SampleRate: 16000, Mode: "transcribe", InputAudioCodec: "pcm_s16le", Default: true}},
		}},
	}
	proxyServer := httptest.NewServer(&Proxy{Config: cfg})
	defer proxyServer.Close()
	wsURL := "ws" + strings.TrimPrefix(proxyServer.URL, "http") + "?provider=sarvam&model=saaras%3Av4&language=en-IN"
	header := http.Header{"Origin": []string{proxyServer.URL}}
	client, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var ready map[string]any
	if err := client.ReadJSON(&ready); err != nil || ready["type"] != "ready" {
		t.Fatalf("ready = %#v, %v", ready, err)
	}
	pcm := []byte{1, 2, 3, 4}
	if err := client.WriteMessage(websocket.BinaryMessage, pcm); err != nil {
		t.Fatal(err)
	}
	if err := client.WriteJSON(map[string]string{"type": "flush"}); err != nil {
		t.Fatal(err)
	}
	audio := <-received
	envelope := audio["audio"].(map[string]any)
	if envelope["data"] != base64.StdEncoding.EncodeToString(pcm) || envelope["encoding"] != "audio/wav" || envelope["sample_rate"] != "16000" {
		t.Fatalf("audio envelope = %#v", envelope)
	}
	if !<-flush {
		t.Fatal("flush was not forwarded unchanged")
	}
	_, message, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var transcript struct {
		Type string `json:"type"`
		Data struct {
			Transcript string `json:"transcript"`
		} `json:"data"`
	}
	if err := json.Unmarshal(message, &transcript); err != nil || transcript.Type != "data" || transcript.Data.Transcript != "hello" {
		t.Fatalf("transcript = %s, %v", message, err)
	}
}

func TestSameOrigin(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.test/voice", nil)
	r.Host = "example.test"
	r.Header.Set("Origin", "https://example.test")
	if !SameOrigin(r) {
		t.Fatal("same host origin rejected")
	}
	r.Header.Set("Origin", "https://evil.test")
	if SameOrigin(r) {
		t.Fatal("cross-origin request accepted")
	}
	r.Host = "localhost:9191"
	r.Header.Set("Origin", "http://localhost:4321")
	if !SameOrigin(r) {
		t.Fatal("loopback development origin rejected")
	}
}
