package browser

import (
	"encoding/json"
	"testing"

	"browser-server/internal/ai/tools"
	corebrowser "browser-server/internal/browser"
)

func toolByName(ts []tools.Tool, name string) tools.Tool {
	for _, tool := range ts {
		if tool.Name == name {
			return tool
		}
	}
	panic("tool not found: " + name)
}

func TestToolSurface(t *testing.T) {
	b := corebrowser.New()
	defer b.Close()
	ts := Tools(&corebrowser.LocalClient{Bus: b})
	if len(ts) == 0 {
		t.Fatal("no tools built")
	}
	names := map[string]bool{}
	for _, tool := range ts {
		if tool.Name == "" || tool.Execute == nil || len(tool.Schema) == 0 {
			t.Fatalf("tool %q incomplete", tool.Name)
		}
		var schema map[string]any
		if err := json.Unmarshal(tool.Schema, &schema); err != nil {
			t.Fatalf("tool %q schema invalid: %v", tool.Name, err)
		}
		names[tool.Name] = true
	}
	for _, want := range []string{
		"browser_list_instances", "browser_list_tabs", "browser_navigate", "browser_click",
		"browser_type", "browser_press", "browser_scroll", "browser_wait", "browser_scrape",
		"browser_eval", "browser_screenshot", "browser_new_tab", "browser_close_tab",
		"browser_focus", "browser_select", "browser_check", "browser_hover", "browser_drag",
		"browser_bring_to_front", "browser_get_cookies", "browser_set_cookie", "browser_pdf",
		"browser_storage", "browser_execute", "browser_stats",
	} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
}

func TestRenderErrorIncludesCandidates(t *testing.T) {
	err := &corebrowser.ResolutionError{Code: corebrowser.ErrCodeTabAmbiguous, Message: "multiple tabs", Candidates: []string{"a", "b"}}
	out := renderError(err)
	if out["ok"] != false {
		t.Fatalf("expected ok=false, got %v", out)
	}
	if out["error_code"] != corebrowser.ErrCodeTabAmbiguous {
		t.Fatalf("expected error_code, got %v", out)
	}
	if _, ok := out["candidates"].([]string); !ok {
		t.Fatalf("expected candidates, got %v", out)
	}
}

func TestExecutorReadOnlyRoundTrip(t *testing.T) {
	b := corebrowser.New()
	defer b.Close()
	_, _ = b.RegisterInstance(corebrowser.Instance{InstanceID: "inst-a", UserID: 1, Browser: "chrome"})
	ts := Tools(&corebrowser.LocalClient{Bus: b})
	res, err := toolByName(ts, "browser_list_instances").Execute(t.Context(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_instances: %v", err)
	}
	m := res.(map[string]any)
	if m["count"] != 1 {
		t.Fatalf("expected 1 instance, got %v", m)
	}
}
