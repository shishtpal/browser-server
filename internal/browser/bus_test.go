package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegisterHeartbeatAndTTL(t *testing.T) {
	now := time.Now()
	b := NewWithClock(func() time.Time { return now })
	defer b.Close()

	inst, err := b.RegisterInstance(Instance{InstanceID: "inst-1", UserID: 1, Browser: "chrome", Label: "Work Chrome"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if !inst.Online {
		t.Fatal("expected online after register")
	}

	// Simulate TTL expiry without a heartbeat.
	now = now.Add(InstanceTTL + time.Second)
	b.sweep()

	list := b.ListInstances()
	if len(list) != 1 || list[0].Online {
		t.Fatalf("expected instance offline after TTL, got %+v", list)
	}

	if err := b.Heartbeat("inst-1"); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	list = b.ListInstances()
	if !list[0].Online {
		t.Fatal("expected online after heartbeat")
	}
}

func TestTargetResolutionBrowser(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	mustRegister(t, b, "inst-b", "Personal Firefox", "firefox")

	// Ambiguous when no browser ref and two online.
	_, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	re, ok := err.(*ResolutionError)
	if !ok || re.Code != ErrCodeBrowserAmbiguous || len(re.Candidates) != 2 {
		t.Fatalf("expected ambiguous with 2 candidates, got %v", err)
	}

	// Label resolution.
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://mail.google.com/", Title: "Inbox", Active: true}}); err != nil {
		t.Fatal(err)
	}
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Browser: &BrowserRef{Label: "Work Chrome"}, Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatalf("label target: %v", err)
	}
	if cmd.InstanceID != "inst-a" || cmd.TabUUID != "tab-1" {
		t.Fatalf("wrong target: %+v", cmd)
	}

	// Unknown label.
	_, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Browser: &BrowserRef{Label: "Nope"}, Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if re, ok := err.(*ResolutionError); !ok || re.Code != ErrCodeBrowserNotFound {
		t.Fatalf("expected browser_not_found, got %v", err)
	}
}

func TestTabGlobResolutionAndAmbiguity(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")

	err := b.SyncTabs("inst-a", []Tab{
		{TabUUID: "tab-1", TabID: 1, URL: "https://mail.google.com/mail/u/0/", Title: "Inbox (2) - a@x.com - Gmail"},
		{TabUUID: "tab-2", TabID: 2, URL: "https://mail.google.com/mail/u/1/", Title: "Inbox - b@x.com - Gmail"},
		{TabUUID: "tab-3", TabID: 3, URL: "https://calendar.google.com/", Title: "Calendar"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Exact URL glob matches one tab.
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{URL: "https://calendar.google.com/*"}},
		Action: ActionClick,
		Params: json.RawMessage(`{"selector":{"css":"button"}}`),
	})
	if err != nil {
		t.Fatalf("single match: %v", err)
	}
	if cmd.TabUUID != "tab-3" {
		t.Fatalf("expected tab-3, got %s", cmd.TabUUID)
	}

	// Ambiguous glob returns candidates.
	_, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{URL: "https://mail.google.com/*"}},
		Action: ActionScrape,
	})
	re, ok := err.(*ResolutionError)
	if !ok || re.Code != ErrCodeTabAmbiguous || len(re.Candidates) != 2 {
		t.Fatalf("expected tab_ambiguous with 2 candidates, got %v", err)
	}

	// Title glob single match.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Title: "Calendar*"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatalf("title match: %v", err)
	}
	if cmd.TabUUID != "tab-3" {
		t.Fatalf("expected tab-3 via title, got %s", cmd.TabUUID)
	}
}

func TestSessionOwnerLock(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	// First session locks the tab.
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target:    TargetRef{Tab: &TabRef{Role: "active"}, SessionID: "task-1"},
		Action:    ActionNavigate,
		Params:    json.RawMessage(`{"url":"https://example.com/page2"}`),
		TimeoutMS: 5000,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Different session targeting the same tab is rejected.
	_, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target:    TargetRef{Tab: &TabRef{Role: "active"}, SessionID: "task-2"},
		Action:    ActionScrape,
		TimeoutMS: 5000,
	})
	re, ok := err.(*ResolutionError)
	if !ok || re.Code != ErrCodeTabBusy {
		t.Fatalf("expected tab_busy, got %v", err)
	}

	// Completing the first command frees the tab.
	if err := b.Result(context.Background(), ResultRequest{CommandID: cmd.CommandID, Status: StatusSucceeded, Result: &CommandResult{}}); err != nil {
		t.Fatal(err)
	}
	_, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target:    TargetRef{Tab: &TabRef{Role: "active"}, SessionID: "task-2"},
		Action:    ActionScrape,
		TimeoutMS: 5000,
	})
	if err != nil {
		t.Fatalf("tab should be free after completion: %v", err)
	}
}

func TestWaitForResult(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan Command, 1)
	go func() {
		res, err := b.WaitForResult(context.Background(), cmd.CommandID, 5*time.Second)
		if err != nil {
			t.Errorf("wait: %v", err)
		}
		done <- res
	}()

	// Simulate the extension completing the command.
	if err := b.Result(context.Background(), ResultRequest{
		CommandID: cmd.CommandID,
		Status:    StatusSucceeded,
		Result:    &CommandResult{Page: &PageInfo{URL: "https://example.com/", Title: "Example"}},
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case res := <-done:
		if res.Status != StatusSucceeded {
			t.Fatalf("expected succeeded, got %s", res.Status)
		}
		if res.Result == nil || res.Result.Page.URL != "https://example.com/" {
			t.Fatalf("bad result: %+v", res.Result)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("waiter never resolved")
	}
}

func TestWaitTimeout(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target:    TargetRef{Tab: &TabRef{Role: "active"}},
		Action:    ActionScrape,
		TimeoutMS: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.WaitForResult(context.Background(), cmd.CommandID, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	got, _ := b.GetCommand(cmd.CommandID)
	if got.Status != StatusTimedOut {
		t.Fatalf("expected timed_out, got %s", got.Status)
	}
}

func TestSSEPushAndQueue(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	events, cancel := b.Subscribe("inst-a")
	defer cancel()

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		if ev.Type != EventCommand || ev.Command == nil || ev.Command.CommandID != cmd.CommandID {
			t.Fatalf("bad event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no SSE event")
	}

	// The alarm fallback path sees the same command in the queue.
	queued := b.Queue("inst-a")
	if len(queued) != 1 || queued[0].CommandID != cmd.CommandID {
		t.Fatalf("queue mismatch: %+v", queued)
	}
}

func TestNewTabRoleAssignsUUID(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "new"}},
		Action: ActionNewTab,
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.TabUUID == "" {
		t.Fatal("expected a generated tab_uuid for role=new")
	}
}

func TestScreenshotResultStoredAsReference(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	// Tiny valid 1x1 transparent PNG.
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	var gotID string
	var gotPNG []byte
	b.SetScreenshotSink(func(ctx context.Context, commandID string, data []byte) (ScreenshotRef, error) {
		gotID, gotPNG = commandID, data
		return ScreenshotRef{
			URL:  "/api/browser/screenshots/" + commandID + ".png",
			Path: filepath.Join("C:\\data", commandID+".png"),
		}, nil
	})

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScreenshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Result(context.Background(), ResultRequest{
		CommandID: cmd.CommandID,
		Status:    StatusSucceeded,
		Result:    &CommandResult{Data: dataURL},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := b.GetCommand(cmd.CommandID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Result == nil {
		t.Fatal("expected a result")
	}
	if want := "/api/browser/screenshots/" + cmd.CommandID + ".png"; stored.Result.Data != want {
		t.Fatalf("expected stored data %q, got %q", want, stored.Result.Data)
	}
	if want := filepath.Join("C:\\data", cmd.CommandID+".png"); stored.Result.Path != want {
		t.Fatalf("expected stored path %q, got %q", want, stored.Result.Path)
	}
	if gotID != cmd.CommandID {
		t.Fatalf("sink command id = %q, want %q", gotID, cmd.CommandID)
	}
	if string(gotPNG) != string(png) {
		t.Fatalf("sink png mismatch: got %d bytes, want %d", len(gotPNG), len(png))
	}
}

func TestPdfResultStoredAsReference(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	// Placeholder PDF payload (the base64 body is not parsed by the bus).
	pdf := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\ntrailer\n<<>>\n%%EOF")
	dataURL := "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(pdf)

	var gotID string
	var gotPDF []byte
	b.SetPdfSink(func(ctx context.Context, commandID string, data []byte) (PdfRef, error) {
		gotID, gotPDF = commandID, data
		return PdfRef{
			URL:  "/api/browser/pdfs/" + commandID + ".pdf",
			Path: filepath.Join("C:\\data", commandID+".pdf"),
		}, nil
	})

	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionPDF,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Result(context.Background(), ResultRequest{
		CommandID: cmd.CommandID,
		Status:    StatusSucceeded,
		Result:    &CommandResult{Data: dataURL},
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := b.GetCommand(cmd.CommandID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Result == nil {
		t.Fatal("expected a result")
	}
	if want := "/api/browser/pdfs/" + cmd.CommandID + ".pdf"; stored.Result.Data != want {
		t.Fatalf("expected stored data %q, got %q", want, stored.Result.Data)
	}
	if want := filepath.Join("C:\\data", cmd.CommandID+".pdf"); stored.Result.Path != want {
		t.Fatalf("expected stored path %q, got %q", want, stored.Result.Path)
	}
	if gotID != cmd.CommandID {
		t.Fatalf("sink command id = %q, want %q", gotID, cmd.CommandID)
	}
	if string(gotPDF) != string(pdf) {
		t.Fatalf("sink pdf mismatch: got %d bytes, want %d", len(gotPDF), len(pdf))
	}
}

func TestNonScreenshotResultKeepsData(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}
	b.SetScreenshotSink(func(ctx context.Context, commandID string, data []byte) (ScreenshotRef, error) {
		t.Fatal("sink must not be called for non-screenshot actions")
		return ScreenshotRef{}, nil
	})

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("abc"))
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionEval,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Result(context.Background(), ResultRequest{
		CommandID: cmd.CommandID,
		Status:    StatusSucceeded,
		Result:    &CommandResult{Data: dataURL},
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := b.GetCommand(cmd.CommandID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Result == nil || stored.Result.Data != dataURL {
		t.Fatalf("expected eval data unchanged, got %+v", stored.Result)
	}
}

func mustRegister(t *testing.T, b *Bus, id, label, browserName string) {
	t.Helper()
	if _, err := b.RegisterInstance(Instance{InstanceID: id, UserID: 1, Browser: browserName, Label: label}); err != nil {
		t.Fatalf("register %s: %v", id, err)
	}
}

func TestCommandLimitsFunc(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://example.com/", Active: true}}); err != nil {
		t.Fatal(err)
	}

	// Package defaults apply when no limits func is installed.
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := int(DefaultCommandTimeout / time.Millisecond); cmd.TimeoutMS != want {
		t.Fatalf("default timeout = %d, want %d", cmd.TimeoutMS, want)
	}

	b.SetCommandLimitsFunc(func() (int, int) { return 30000, 90000 })

	// Missing timeout_ms uses the configured default.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.TimeoutMS != 30000 {
		t.Fatalf("default timeout = %d, want 30000", cmd.TimeoutMS)
	}

	// Timeouts above the configured maximum are clamped.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target:    TargetRef{Tab: &TabRef{Role: "active"}},
		Action:    ActionScrape,
		TimeoutMS: 120000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.TimeoutMS != 90000 {
		t.Fatalf("clamped timeout = %d, want 90000", cmd.TimeoutMS)
	}
}

func TestEvalModeInjection(t *testing.T) {
	b := NewWithClock(func() time.Time { return time.Now() })
	defer b.Close()
	mustRegister(t, b, "inst-a", "Work Chrome", "chrome")
	if err := b.SyncTabs("inst-a", []Tab{{TabUUID: "tab-1", TabID: 1, URL: "https://www.youtube.com/watch?v=abc", Active: true}}); err != nil {
		t.Fatal(err)
	}

	b.SetEvalModeFunc(func(tabURL string) string {
		if strings.Contains(tabURL, "youtube.com") {
			return "cdp"
		}
		return "inject"
	})

	// Eval without an explicit mode gets the resolved mode injected.
	cmd, err := b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionEval,
		Params: json.RawMessage(`{"expression":"document.title"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramsString(t, cmd.Params, "mode"); got != "cdp" {
		t.Fatalf("injected mode = %q, want cdp", got)
	}

	// An explicit mode always wins over the resolver.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionEval,
		Params: json.RawMessage(`{"expression":"document.title","mode":"inject"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramsString(t, cmd.Params, "mode"); got != "inject" {
		t.Fatalf("explicit mode = %q, want inject", got)
	}

	// Orchestrate (browser_execute) inherits the mode too.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionOrchestrate,
		Params: json.RawMessage(`{"steps":[{"action":"eval","params":{"expression":"1+1"}}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramsString(t, cmd.Params, "mode"); got != "cdp" {
		t.Fatalf("orchestrate mode = %q, want cdp", got)
	}

	// Non-eval actions are never touched.
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionScrape,
		Params: json.RawMessage(`{"extract":{"title":"h1"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramsString(t, cmd.Params, "mode"); got != "" {
		t.Fatalf("scrape should have no mode, got %q", got)
	}

	// A resolver returning "" leaves the params untouched.
	b.SetEvalModeFunc(func(string) string { return "" })
	cmd, err = b.CreateCommand(context.Background(), CreateCommandRequest{
		Target: TargetRef{Tab: &TabRef{Role: "active"}},
		Action: ActionEval,
		Params: json.RawMessage(`{"expression":"document.title"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := paramsString(t, cmd.Params, "mode"); got != "" {
		t.Fatalf("empty resolver should not inject a mode, got %q", got)
	}
}

func paramsString(t *testing.T, raw json.RawMessage, key string) string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		t.Fatalf("param %q is not a string: %v", key, err)
	}
	return s
}
