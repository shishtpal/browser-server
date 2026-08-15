package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"browser-server/internal/browser"
	"browser-server/internal/helpers"
)

// BrowserBus is the shared in-memory browser command bus. The AI module owns
// the instance; main wires it here so the REST endpoints and the AI tools see
// the same state. Nil means browser automation is not available.
var BrowserBus *browser.Bus

// BrowserScreenshotDir is the directory holding saved browser screenshot PNGs.
// The AI module populates it at startup; BrowserScreenshotFile serves them.
var BrowserScreenshotDir string

// BrowserPdfDir is the directory holding saved browser PDFs. The AI module
// populates it at startup; BrowserPdfFile serves them.
var BrowserPdfDir string

func browserBusOr503(w http.ResponseWriter) *browser.Bus {
	if BrowserBus == nil {
		helpers.WriteError(w, http.StatusServiceUnavailable, "browser automation is not enabled")
		return nil
	}
	return BrowserBus
}

// BrowserRegister handles the extension's startup registration.
func BrowserRegister(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	var req browser.Instance
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	inst, err := bus.RegisterInstance(req)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, inst)
}

// BrowserHeartbeat refreshes an instance's online TTL.
func BrowserHeartbeat(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	var req struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := bus.Heartbeat(req.InstanceID); err != nil {
		helpers.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// BrowserTabs receives a tab registry snapshot or incremental delta from the
// extension. A JSON boolean patch=true switches to delta mode.
func BrowserTabs(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	var req struct {
		InstanceID string        `json:"instance_id"`
		Patch      bool          `json:"patch"`
		Tabs       []browser.Tab `json:"tabs"`
		Removed    []string      `json:"removed"` // tab_uuid list (delta mode)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if req.InstanceID == "" {
		helpers.WriteError(w, http.StatusBadRequest, "instance_id is required")
		return
	}
	if req.Patch {
		for _, uuid := range req.Removed {
			bus.RemoveTab(req.InstanceID, uuid)
		}
		if err := bus.PatchTabs(req.InstanceID, req.Tabs); err != nil {
			helpers.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
	} else {
		if err := bus.SyncTabs(req.InstanceID, req.Tabs); err != nil {
			helpers.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// BrowserListInstances returns all registered browsers, online first.
func BrowserListInstances(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	helpers.WriteJSON(w, http.StatusOK, bus.ListInstances())
}

// BrowserListTabs returns the tab registry for one instance.
func BrowserListTabs(w http.ResponseWriter, r *http.Request, instanceID string) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	tabs, err := bus.ListTabs(instanceID)
	if err != nil {
		writeBrowserResolutionError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, tabs)
}

// BrowserCreateCommand is the AI-side entry point (web chat in-process tools
// may bypass it via the local client; the CLI uses it over HTTP).
func BrowserCreateCommand(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	var req browser.CreateCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	cmd, err := bus.CreateCommand(r.Context(), req)
	if err != nil {
		writeBrowserResolutionError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, cmd)
}

// BrowserGetCommand returns the current command state (CLI polling path).
func BrowserGetCommand(w http.ResponseWriter, r *http.Request, commandID string) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	cmd, err := bus.GetCommand(commandID)
	if err != nil {
		writeBrowserResolutionError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, cmd)
}

// BrowserScreenshotFile serves a stored browser screenshot PNG by filename.
// Filenames are <command_id>.png (bus-generated "cmd_<hex>"); the name is
// strictly validated so the route cannot escape the screenshot directory.
func BrowserScreenshotFile(w http.ResponseWriter, r *http.Request, filename string) {
	if BrowserScreenshotDir == "" || !safeScreenshotName(filename) {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(filepath.Join(BrowserScreenshotDir, filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, filename, time.Time{}, f)
}

// safeScreenshotName restricts filenames to alphanumerics, dash, underscore
// plus a ".png" suffix (mirrors the naming in the AI browser harness).
func safeScreenshotName(name string) bool {
	if !strings.HasSuffix(name, ".png") {
		return false
	}
	for _, c := range strings.TrimSuffix(name, ".png") {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// BrowserPdfFile serves a stored browser PDF by filename. Filenames are
// <command_id>.pdf (bus-generated "cmd_<hex>"); the name is strictly validated
// so the route cannot escape the PDF directory.
func BrowserPdfFile(w http.ResponseWriter, r *http.Request, filename string) {
	if BrowserPdfDir == "" || !safePdfName(filename) {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(filepath.Join(BrowserPdfDir, filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/pdf")
	http.ServeContent(w, r, filename, time.Time{}, f)
}

// safePdfName restricts filenames to alphanumerics, dash, underscore plus a
// ".pdf" suffix (mirrors the naming in the AI browser harness).
func safePdfName(name string) bool {
	if !strings.HasSuffix(name, ".pdf") {
		return false
	}
	for _, c := range strings.TrimSuffix(name, ".pdf") {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// BrowserResult receives the extension's processing ack or final result.
func BrowserResult(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	var req browser.ResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := bus.Result(r.Context(), req); err != nil {
		writeBrowserResolutionError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// BrowserQueue returns queued commands for an instance — the extension's SSE
// reconnect / alarm fallback.
func BrowserQueue(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		helpers.WriteError(w, http.StatusBadRequest, "instance_id is required")
		return
	}
	helpers.WriteJSON(w, http.StatusOK, bus.Queue(instanceID))
}

// BrowserEvents is the SSE push channel for one extension instance. The
// extension connects with its instance_id; commands and pings flow here.
func BrowserEvents(w http.ResponseWriter, r *http.Request) {
	bus := browserBusOr503(w)
	if bus == nil {
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		helpers.WriteError(w, http.StatusBadRequest, "instance_id is required")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		helpers.WriteError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	events, cancel := bus.Subscribe(instanceID)
	defer cancel()

	// Immediate queue replay so a freshly connected extension never misses
	// commands enqueued while it was offline.
	queued := bus.Queue(instanceID)
	for i := range queued {
		writeBrowserSSE(w, flusher, browser.Event{Type: browser.EventCommand, Command: &queued[i]})
	}

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	closed := r.Context().Done()
	for {
		select {
		case <-closed:
			return
		case ev, open := <-events:
			if !open {
				return
			}
			writeBrowserSSE(w, flusher, ev)
		case <-keepalive.C:
			writeBrowserSSE(w, flusher, browser.Event{Type: browser.EventPing})
		}
	}
}

func writeBrowserSSE(w http.ResponseWriter, flusher http.Flusher, ev browser.Event) {
	data, _ := json.Marshal(ev)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
	flusher.Flush()
}

// writeBrowserResolutionError renders a ResolutionError with its candidate
// list, or a generic 500 for unexpected failures.
func writeBrowserResolutionError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if re, ok := err.(*browser.ResolutionError); ok {
		body := map[string]any{"error": re.Message, "code": re.Code}
		if len(re.Candidates) > 0 {
			body["candidates"] = re.Candidates
		}
		helpers.WriteJSON(w, status, body)
		return
	}
	helpers.WriteError(w, http.StatusInternalServerError, err.Error())
}
