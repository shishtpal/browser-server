package voice

import (
	"encoding/base64"
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
