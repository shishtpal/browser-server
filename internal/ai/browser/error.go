package browser

import (
	"errors"

	corebrowser "browser-server/internal/browser"
)

// renderError converts a bus/transport error into a structured result so the
// model can self-correct (candidates are included for ambiguity).
func renderError(err error) map[string]any {
	if err == nil {
		return map[string]any{"ok": true}
	}
	out := map[string]any{"ok": false, "error": err.Error()}
	if code := corebrowser.ResolutionCode(err); code != "" {
		out["error_code"] = code
	}
	var re *corebrowser.ResolutionError
	if errors.As(err, &re) && len(re.Candidates) > 0 {
		out["candidates"] = re.Candidates
		out["next_step"] = "Call browser_list_instances or browser_list_tabs to disambiguate, then retry with an exact instance_id/tab.uuid."
	}
	return out
}
