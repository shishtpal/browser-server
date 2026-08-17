package voice

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Proxy struct {
	Config *Config
	Dialer *websocket.Dialer
}

func BuildUpstreamURL(s Selection) (string, error) {
	u, err := url.Parse(s.Provider.BaseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("model", s.Model.ID)
	q.Set("language-code", s.Language.Code)
	q.Set("mode", s.Model.Mode)
	q.Set("sample_rate", fmt.Sprint(s.Model.SampleRate))
	q.Set("input_audio_codec", s.Model.InputAudioCodec)
	q.Set("flush_signal", "true")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func SameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	requestHost := r.Host
	if host, _, splitErr := net.SplitHostPort(r.Host); splitErr == nil {
		requestHost = host
	}
	// Allow any loopback origin only when the server itself is also on loopback.
	// This keeps development simple without weakening production deployments.
	return isLoopback(u.Hostname()) && isLoopback(requestHost)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s, err := p.Config.Select(r.URL.Query().Get("provider"), r.URL.Query().Get("model"), r.URL.Query().Get("language"))
	if err != nil {
		http.Error(w, "invalid voice selection", http.StatusBadRequest)
		return
	}
	if s.Provider.IsOpenRouterSTT() {
		p.serveBatchSTT(w, r, s)
		return
	}
	p.serveStreaming(w, r, s)
}

// serveStreaming handles the Sarvam WebSocket streaming proxy path.
func (p *Proxy) serveStreaming(w http.ResponseWriter, r *http.Request, s Selection) {
	upstreamURL, err := BuildUpstreamURL(s)
	if err != nil {
		http.Error(w, "voice service unavailable", http.StatusServiceUnavailable)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: SameOrigin, ReadBufferSize: 4096, WriteBufferSize: 4096}
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer client.Close()

	timeout := time.Duration(s.Provider.RequestTimeoutSeconds) * time.Second
	dialer := p.Dialer
	if dialer == nil {
		dialer = &websocket.Dialer{HandshakeTimeout: min(timeout, 15*time.Second)}
	}
	header := http.Header{"Api-Subscription-Key": []string{s.Provider.APIKey}}
	upstream, _, err := dialer.Dial(upstreamURL, header)
	if err != nil {
		writeClientJSON(client, nil, map[string]any{"type": "error", "message": "voice service unavailable"})
		return
	}
	defer upstream.Close()

	sessionLimit := time.Duration(p.Config.Recording.MaxDurationSecs)*time.Second + 15*time.Second
	deadline := time.Now().Add(min(timeout, sessionLimit))
	client.SetReadDeadline(deadline)
	client.SetWriteDeadline(deadline)
	client.SetReadLimit(p.Config.Recording.MaxFrameBytes)
	upstream.SetReadDeadline(deadline)
	upstream.SetWriteDeadline(deadline)
	var writeMu sync.Mutex
	if err := writeClientJSON(client, &writeMu, map[string]string{"type": "ready"}); err != nil {
		return
	}
	errCh := make(chan error, 2)
	go p.browserToUpstream(client, upstream, s, errCh)
	go upstreamToBrowser(upstream, client, &writeMu, errCh)
	err = <-errCh
	if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		_ = writeClientJSON(client, &writeMu, map[string]any{"type": "error", "message": clientError(err)})
	}
	writeMu.Lock()
	_ = client.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	writeMu.Unlock()
}

// serveBatchSTT handles the OpenRouter batch STT path: accumulates PCM frames
// from the browser, then on flush sends the complete audio as a single
// request to the OpenRouter /audio/transcriptions endpoint.
func (p *Proxy) serveBatchSTT(w http.ResponseWriter, r *http.Request, s Selection) {
	upgrader := websocket.Upgrader{CheckOrigin: SameOrigin, ReadBufferSize: 4096, WriteBufferSize: 4096}
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer client.Close()

	var writeMu sync.Mutex
	if err := writeClientJSON(client, &writeMu, map[string]string{"type": "ready"}); err != nil {
		return
	}

	var audioBuf bytes.Buffer
	var totalBytes int64
	sampleRate := s.Model.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	deadline := time.Now().Add(time.Duration(p.Config.Recording.MaxDurationSecs)*time.Second + 30*time.Second)
	client.SetReadDeadline(deadline)
	client.SetWriteDeadline(deadline)
	client.SetReadLimit(p.Config.Recording.MaxFrameBytes)

	for {
		messageType, data, err := client.ReadMessage()
		if err != nil {
			return
		}
		switch messageType {
		case websocket.BinaryMessage:
			totalBytes += int64(len(data))
			if int64(len(data)) > p.Config.Recording.MaxFrameBytes || totalBytes > p.Config.Recording.MaxAudioBytes {
				_ = writeClientJSON(client, &writeMu, map[string]any{"type": "error", "message": "audio limit exceeded"})
				return
			}
			audioBuf.Write(data)
		case websocket.TextMessage:
			var control struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &control) != nil || control.Type != "flush" {
				_ = writeClientJSON(client, &writeMu, map[string]any{"type": "error", "message": "invalid control message"})
				return
			}
			// Build WAV from accumulated PCM and send to OpenRouter.
			wav := buildWAV(audioBuf.Bytes(), sampleRate, 1, 16)
			result, err := Transcribe(r.Context(), p.Config, STTRequest{
				AudioBytes:  wav,
				AudioFormat: "wav",
				ProviderID:  s.ProviderID,
				ModelID:     s.Model.ID,
				Language:    s.Language.Code,
			})
			if err != nil {
				msg := "transcription failed"
				if errors.Is(err, ErrProvider) {
					msg = "transcription provider error: " + err.Error()
				}
				_ = writeClientJSON(client, &writeMu, map[string]any{"type": "error", "message": msg})
				return
			}
			// Send the transcript in the same envelope format the frontend expects.
			_ = writeClientJSON(client, &writeMu, map[string]any{
				"type": "data",
				"data": map[string]string{"transcript": result.Text},
			})
			writeMu.Lock()
			_ = client.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			writeMu.Unlock()
			return
		default:
			_ = writeClientJSON(client, &writeMu, map[string]any{"type": "error", "message": "unsupported websocket frame"})
			return
		}
	}
}

func (p *Proxy) browserToUpstream(client, upstream *websocket.Conn, s Selection, done chan<- error) {
	var total int64
	for {
		messageType, data, err := client.ReadMessage()
		if err != nil {
			done <- err
			return
		}
		switch messageType {
		case websocket.BinaryMessage:
			total += int64(len(data))
			if int64(len(data)) > p.Config.Recording.MaxFrameBytes || total > p.Config.Recording.MaxAudioBytes {
				done <- errors.New("audio limit exceeded")
				return
			}
			payload := map[string]any{"audio": map[string]any{"data": base64.StdEncoding.EncodeToString(data), "encoding": "audio/wav", "sample_rate": fmt.Sprint(s.Model.SampleRate)}}
			if err := upstream.WriteJSON(payload); err != nil {
				done <- err
				return
			}
		case websocket.TextMessage:
			var control struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &control) != nil || control.Type != "flush" {
				done <- errors.New("invalid control message")
				return
			}
			if err := upstream.WriteMessage(websocket.TextMessage, []byte(`{"type":"flush"}`)); err != nil {
				done <- err
				return
			}
		default:
			done <- errors.New("unsupported websocket frame")
			return
		}
	}
}

func upstreamToBrowser(upstream, client *websocket.Conn, mu *sync.Mutex, done chan<- error) {
	for {
		messageType, data, err := upstream.ReadMessage()
		if err != nil {
			done <- err
			return
		}
		if messageType != websocket.TextMessage || !json.Valid(data) {
			done <- errors.New("invalid voice service response")
			return
		}
		var envelope struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			done <- errors.New("invalid voice service response")
			return
		}
		if envelope.Type == "error" {
			done <- errors.New("voice service error")
			return
		}
		if envelope.Type != "data" && envelope.Type != "events" {
			done <- errors.New("unexpected voice service response")
			return
		}
		mu.Lock()
		err = client.WriteMessage(websocket.TextMessage, data)
		mu.Unlock()
		if err != nil {
			done <- err
			return
		}
	}
}

func writeClientJSON(c *websocket.Conn, mu *sync.Mutex, value any) error {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	return c.WriteJSON(value)
}

func clientError(err error) string {
	if strings.Contains(err.Error(), "limit") {
		return "audio limit exceeded"
	}
	if strings.Contains(err.Error(), "control") || strings.Contains(err.Error(), "frame") {
		return "invalid voice message"
	}
	return "voice transcription failed"
}

// buildWAV prepends a standard 44-byte WAV header to raw PCM16 mono data.
func buildWAV(pcm []byte, sampleRate, numChannels, bitsPerSample int) []byte {
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(pcm)
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // PCM chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM format
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))
	out := make([]byte, 44+dataSize)
	copy(out, header)
	copy(out[44:], pcm)
	return out
}
