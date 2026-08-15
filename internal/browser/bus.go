package browser

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Bus is the in-memory command channel. It is the single source of truth for
// browser instances, tabs, and commands while the server process is alive.
//
// Concurrency: all maps are guarded by one mutex. SSE subscribers and command
// waiters are signaled via channels after the lock is released, so no handler
// ever sends on a channel while holding mu.
type Bus struct {
	mu             sync.Mutex
	instances      map[string]*Instance
	tabs           map[string]map[string]*Tab // instanceID -> tabUUID -> tab
	commands       map[string]*Command
	perInstance    []string                           // command ids in insertion order, bounded globally
	owners         map[tabKey]string                  // tab_key -> owning session
	waiters        map[string]chan *Command           // commandID -> result channel
	subs           map[string]map[chan Event]struct{} // instanceID -> subscriber chans
	now            func() time.Time
	started        bool
	stop           chan struct{}
	wg             sync.WaitGroup
	screenshotSink ScreenshotSink
	pdfSink        PdfSink
	evalMode       EvalModeFunc
	commandLimits  CommandLimitsFunc
	stats          struct {
		succeeded int64
		failed    int64
		timedOut  int64
		total     int64
	}
}

// ScreenshotRef describes a persisted screenshot: the server-relative URL that
// serves it back plus the local filesystem path of the stored file.
type ScreenshotRef struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

// ScreenshotSink persists a screenshot PNG payload and returns a reference
// (serving URL + local file path). When installed, a succeeded screenshot
// command replaces its inline base64 data URL with the returned URL and records
// the file path on the result, so tool output stays small and downstream tools
// (e.g. ocr_image) can read the file directly. It may be nil, in which case
// screenshot results stay inline.
type ScreenshotSink func(ctx context.Context, commandID string, png []byte) (ScreenshotRef, error)

// SetScreenshotSink installs the screenshot persistence hook. Safe to call
// before or after commands are issued.
func (b *Bus) SetScreenshotSink(sink ScreenshotSink) {
	b.mu.Lock()
	b.screenshotSink = sink
	b.mu.Unlock()
}

// PdfRef describes a persisted browser PDF: the server-relative URL that serves
// it back plus the local filesystem path of the stored file.
type PdfRef struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

// PdfSink persists a PDF payload and returns a reference (serving URL + local
// file path). Mirrors ScreenshotSink for the pdf action, so a succeeded pdf
// command replaces its inline base64 data URL with the returned URL and records
// the file path on the result. It may be nil, in which case pdf results stay
// inline.
type PdfSink func(ctx context.Context, commandID string, pdf []byte) (PdfRef, error)

// SetPdfSink installs the PDF persistence hook. Safe to call before or after
// commands are issued.
func (b *Bus) SetPdfSink(sink PdfSink) {
	b.mu.Lock()
	b.pdfSink = sink
	b.mu.Unlock()
}

// EvalModeFunc resolves the effective eval execution mode for a resolved tab
// URL. It is consulted for eval and orchestrate commands whose params omit an
// explicit "mode". Return "" to leave the mode untouched (extension default).
// The AI module installs this from bs-browser-config.json so the bus stays
// config-agnostic.
type EvalModeFunc func(tabURL string) string

// CommandLimitsFunc returns the configured per-command timeout bounds in
// milliseconds (default, then hard maximum). It replaces the package defaults
// when set, so bs-browser-config.json can tune both limits at runtime and the
// admin editor changes apply without a restart.
type CommandLimitsFunc func() (defaultMS, maxMS int)

// SetEvalModeFunc installs the eval-mode resolver. Safe to call any time.
func (b *Bus) SetEvalModeFunc(fn EvalModeFunc) {
	b.mu.Lock()
	b.evalMode = fn
	b.mu.Unlock()
}

// SetCommandLimitsFunc installs the command timeout bounds resolver. Safe to
// call any time.
func (b *Bus) SetCommandLimitsFunc(fn CommandLimitsFunc) {
	b.mu.Lock()
	b.commandLimits = fn
	b.mu.Unlock()
}

// CommandStats returns a snapshot of the bus's command counters.
func (b *Bus) CommandStats() map[string]int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return map[string]int64{
		"total":     b.stats.total,
		"succeeded": b.stats.succeeded,
		"failed":    b.stats.failed,
		"timed_out": b.stats.timedOut,
	}
}

// New creates a Bus and starts its background sweeper. Call Close to stop it.
func New() *Bus {
	b := &Bus{
		instances: make(map[string]*Instance),
		tabs:      make(map[string]map[string]*Tab),
		commands:  make(map[string]*Command),
		waiters:   make(map[string]chan *Command),
		subs:      make(map[string]map[chan Event]struct{}),
		now:       time.Now,
		stop:      make(chan struct{}),
	}
	b.start()
	return b
}

// NewWithClock creates a Bus with an injectable clock (tests).
func NewWithClock(now func() time.Time) *Bus {
	b := New()
	b.now = now
	return b
}

// Close stops background sweepers.
func (b *Bus) Close() {
	b.mu.Lock()
	if b.started {
		close(b.stop)
		b.started = false
	}
	b.mu.Unlock()
	b.wg.Wait()
}

func (b *Bus) start() {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return
	}
	b.started = true
	stop := b.stop
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.sweep()
			case <-stop:
				return
			}
		}
	}()
}

func (b *Bus) nowTime() time.Time {
	if b.now != nil {
		return b.now()
	}
	return time.Now()
}

func newID(prefix string) string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}

// ---------------------------------------------------------------------------
// Instances
// ---------------------------------------------------------------------------

// RegisterInstance upserts an extension instance and marks it online.
func (b *Bus) RegisterInstance(req Instance) (Instance, error) {
	if strings.TrimSpace(req.InstanceID) == "" {
		return Instance{}, &ResolutionError{Code: ErrCodeInvalidTarget, Message: "instance_id is required"}
	}
	if req.UserID <= 0 {
		req.UserID = 1
	}
	if req.Browser == "" {
		req.Browser = "unknown"
	}
	now := b.nowTime()
	b.mu.Lock()
	existing, ok := b.instances[req.InstanceID]
	if ok {
		existing.Browser = req.Browser
		existing.Version = req.Version
		if req.Label != "" {
			existing.Label = req.Label
		}
		existing.UserID = req.UserID
		existing.Online = true
		existing.LastSeenAt = now
		b.mu.Unlock()
		return *existing, nil
	}
	inst := req
	inst.Online = true
	inst.LastSeenAt = now
	inst.CreatedAt = now
	b.instances[req.InstanceID] = &inst
	b.mu.Unlock()
	return inst, nil
}

// Heartbeat marks an instance online and refreshes its TTL.
func (b *Bus) Heartbeat(instanceID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	inst, ok := b.instances[instanceID]
	if !ok {
		return &ResolutionError{Code: ErrCodeInstanceNotRegistered, Message: "instance not registered; run the extension and reload it"}
	}
	inst.Online = true
	inst.LastSeenAt = b.nowTime()
	return nil
}

// ListInstances returns all instances, online first.
func (b *Bus) ListInstances() []Instance {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Instance, 0, len(b.instances))
	for _, inst := range b.instances {
		out = append(out, *inst)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Online != out[j].Online {
			return out[i].Online
		}
		return out[i].LastSeenAt.After(out[j].LastSeenAt)
	})
	return out
}

// ---------------------------------------------------------------------------
// Tabs
// ---------------------------------------------------------------------------

// SyncTabs replaces the tab registry for an instance with a full snapshot
// (the extension sends its current tab list on startup and on every change).
func (b *Bus) SyncTabs(instanceID string, tabs []Tab) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	inst, ok := b.instances[instanceID]
	if !ok {
		return &ResolutionError{Code: ErrCodeInstanceNotRegistered, Message: "instance not registered"}
	}
	now := b.nowTime()
	next := make(map[string]*Tab, len(tabs))
	for i := range tabs {
		t := tabs[i]
		t.InstanceID = instanceID
		if t.UpdatedAt == nil || t.UpdatedAt.IsZero() {
			t.UpdatedAt = &now
		}
		if (t.LastActiveAt == nil || t.LastActiveAt.IsZero()) && t.Active {
			t.LastActiveAt = &now
		}
		next[t.TabUUID] = &t
	}
	// Preserve active state from the incoming snapshot verbatim; any tab not
	// present in the snapshot is treated as closed.
	b.tabs[instanceID] = next
	inst.LastSeenAt = now
	inst.Online = true
	return nil
}

// PatchTabs applies an incremental delta (add/update, and removal for entries
// with a Closed flag).
func (b *Bus) PatchTabs(instanceID string, tabs []Tab) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	inst, ok := b.instances[instanceID]
	if !ok {
		return &ResolutionError{Code: ErrCodeInstanceNotRegistered, Message: "instance not registered"}
	}
	if b.tabs[instanceID] == nil {
		b.tabs[instanceID] = make(map[string]*Tab)
	}
	now := b.nowTime()
	for i := range tabs {
		t := tabs[i]
		t.InstanceID = instanceID
		if t.TabUUID == "" {
			continue
		}
		if t.UpdatedAt == nil || t.UpdatedAt.IsZero() {
			t.UpdatedAt = &now
		}
		if (t.LastActiveAt == nil || t.LastActiveAt.IsZero()) && t.Active {
			t.LastActiveAt = &now
		}
		b.tabs[instanceID][t.TabUUID] = &t
	}
	inst.LastSeenAt = now
	inst.Online = true
	return nil
}

// RemoveTab drops a closed tab.
func (b *Bus) RemoveTab(instanceID, tabUUID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.tabs[instanceID]; ok {
		delete(m, tabUUID)
	}
}

// ListTabs returns tabs for an instance sorted by last-active descending.
func (b *Bus) ListTabs(instanceID string) ([]Tab, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.instances[instanceID]; !ok {
		return nil, &ResolutionError{Code: ErrCodeBrowserNotFound, Message: fmt.Sprintf("browser %q not found", instanceID)}
	}
	m := b.tabs[instanceID]
	out := make([]Tab, 0, len(m))
	for _, t := range m {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool {
		li := derefTime(out[i].LastActiveAt)
		lj := derefTime(out[j].LastActiveAt)
		return li.After(lj)
	})
	return out, nil
}

// derefTime safely dereferences a *time.Time, returning zero time for nil.
func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

// CreateCommand validates the target, binds a tab, enqueues the command, and
// pushes it to the instance's SSE channel. It returns the created command or
// a ResolutionError carrying candidate lists.
func (b *Bus) CreateCommand(ctx context.Context, req CreateCommandRequest) (Command, error) {
	if strings.TrimSpace(req.Action) == "" {
		return Command{}, &ResolutionError{Code: ErrCodeUnknownAction, Message: "action is required"}
	}
	if !validAction(req.Action) {
		return Command{}, &ResolutionError{Code: ErrCodeUnknownAction, Message: fmt.Sprintf("unknown action %q", req.Action)}
	}
	timeout := req.TimeoutMS
	defaultMS, maxMS := b.limits()
	if timeout <= 0 {
		timeout = defaultMS
	}
	if timeout > maxMS {
		timeout = maxMS
	}
	session := strings.TrimSpace(req.Target.SessionID)
	if session == "" {
		session = DefaultSessionID
	}

	instanceID, tabUUID, err := b.resolveTargetLocked(req.Target, session)
	if err != nil {
		return Command{}, err
	}

	cmd := Command{
		CommandID:  newID("cmd"),
		InstanceID: instanceID,
		TabUUID:    tabUUID,
		SessionID:  session,
		Action:     req.Action,
		Params:     req.Params,
		Status:     StatusQueued,
		TimeoutMS:  timeout,
		CreatedAt:  b.nowTime(),
	}
	b.setOwner(instanceID, tabUUID, session)
	if len(cmd.Params) == 0 || string(cmd.Params) == "null" {
		cmd.Params = json.RawMessage("{}")
	}
	// A "new" role tab is not in the registry yet, so its URL is unknown;
	// the resolver falls back to the configured global default.
	b.injectEvalModeLocked(&cmd, tabURLOf(b.tabs[instanceID][tabUUID]))
	b.enqueueLocked(&cmd)
	b.pushLocked(instanceID, Event{Type: EventCommand, Command: &cmd})
	return cmd, nil
}

// limits returns the effective per-command timeout bounds in milliseconds,
// from the installed CommandLimitsFunc or the package defaults.
func (b *Bus) limits() (int, int) {
	defaultMS := int(DefaultCommandTimeout / time.Millisecond)
	maxMS := int(MaxCommandTimeout / time.Millisecond)
	if b.commandLimits != nil {
		if d, m := b.commandLimits(); d > 0 && m > 0 {
			return d, m
		}
	}
	return defaultMS, maxMS
}

// tabURLOf returns a tab's URL, or "" when the tab is not yet in the registry.
func tabURLOf(t *Tab) string {
	if t == nil {
		return ""
	}
	return t.URL
}

// injectEvalModeLocked defaults the "mode" param of eval and orchestrate
// commands from the installed EvalModeFunc whenever the params omit it. This
// lets an operator force CDP execution on CSP-strict domains without the
// model requesting it. It is a no-op when no resolver is installed or the
// mode is already set.
func (b *Bus) injectEvalModeLocked(cmd *Command, tabURL string) {
	if b.evalMode == nil || len(cmd.Params) == 0 {
		return
	}
	if cmd.Action != ActionEval && cmd.Action != ActionOrchestrate {
		return
	}
	var params map[string]any
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return
	}
	if _, exists := params["mode"]; exists {
		return
	}
	if mode := b.evalMode(tabURL); mode != "" {
		params["mode"] = mode
		encoded, err := json.Marshal(params)
		if err != nil {
			return
		}
		cmd.Params = encoded
	}
}

// resolveTargetLocked must be called with b.mu held.
func (b *Bus) resolveTargetLocked(t TargetRef, session string) (string, string, error) {
	instanceID, err := b.resolveBrowserLocked(t.Browser)
	if err != nil {
		return "", "", err
	}
	tabUUID, err := b.resolveTabLocked(instanceID, t.Tab, session)
	if err != nil {
		return "", "", err
	}
	return instanceID, tabUUID, nil
}

func (b *Bus) resolveBrowserLocked(ref *BrowserRef) (string, error) {
	online := make([]Instance, 0)
	for _, inst := range b.instances {
		if inst.Online {
			online = append(online, *inst)
		}
	}
	if ref == nil {
		if len(online) == 1 {
			return online[0].InstanceID, nil
		}
		if len(online) == 0 {
			return "", &ResolutionError{Code: ErrCodeNoBrowser, Message: "no browser instance is online; open the extension in a browser first"}
		}
		cands := make([]string, len(online))
		for i, inst := range online {
			cands[i] = instanceSummary(inst)
		}
		return "", &ResolutionError{Code: ErrCodeBrowserAmbiguous, Message: "multiple browsers are online; specify browser.instance_id or browser.label", Candidates: cands}
	}
	if ref.InstanceID != "" {
		inst, ok := b.instances[ref.InstanceID]
		if !ok {
			return "", &ResolutionError{Code: ErrCodeBrowserNotFound, Message: fmt.Sprintf("browser %q not found", ref.InstanceID)}
		}
		if !inst.Online {
			return "", &ResolutionError{Code: ErrCodeBrowserOffline, Message: fmt.Sprintf("browser %q is offline", ref.InstanceID)}
		}
		return inst.InstanceID, nil
	}
	if ref.Label != "" {
		matches := make([]Instance, 0)
		for _, inst := range b.instances {
			if inst.Online && strings.EqualFold(inst.Label, ref.Label) {
				matches = append(matches, *inst)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0].InstanceID, nil
		case 0:
			return "", &ResolutionError{Code: ErrCodeBrowserNotFound, Message: fmt.Sprintf("no online browser labeled %q", ref.Label)}
		default:
			cands := make([]string, len(matches))
			for i, inst := range matches {
				cands[i] = instanceSummary(inst)
			}
			return "", &ResolutionError{Code: ErrCodeBrowserAmbiguous, Message: fmt.Sprintf("multiple browsers share label %q", ref.Label), Candidates: cands}
		}
	}
	if ref.FirstOnline {
		if len(online) == 1 {
			return online[0].InstanceID, nil
		}
		if len(online) == 0 {
			return "", &ResolutionError{Code: ErrCodeNoBrowser, Message: "no browser instance is online"}
		}
		cands := make([]string, len(online))
		for i, inst := range online {
			cands[i] = instanceSummary(inst)
		}
		return "", &ResolutionError{Code: ErrCodeBrowserAmbiguous, Message: "first_online requires exactly one online browser", Candidates: cands}
	}
	return "", &ResolutionError{Code: ErrCodeInvalidTarget, Message: "browser target must set instance_id, label, or first_online"}
}

// resolveTabLocked resolves the tab reference and enforces the per-tab session
// owner lock. Must be called with b.mu held.
func (b *Bus) resolveTabLocked(instanceID string, ref *TabRef, session string) (string, error) {
	tabs := b.tabs[instanceID]
	if ref == nil {
		return "", &ResolutionError{Code: ErrCodeInvalidTarget, Message: "tab target is required (uuid, url, title, or role)"}
	}
	if ref.UUID != "" {
		tab, ok := tabs[ref.UUID]
		if !ok {
			return "", &ResolutionError{Code: ErrCodeTabNotFound, Message: fmt.Sprintf("tab %q not found; list tabs with browser_list_tabs", ref.UUID)}
		}
		if err := b.checkOwnerLocked(instanceID, tab.TabUUID, session); err != nil {
			return "", err
		}
		return tab.TabUUID, nil
	}
	if ref.URL != "" || ref.Title != "" {
		matches := make([]Tab, 0)
		for _, t := range tabs {
			if ref.URL != "" && !globMatch(ref.URL, t.URL) {
				continue
			}
			if ref.Title != "" && !globMatch(ref.Title, t.Title) {
				continue
			}
			matches = append(matches, *t)
		}
		switch len(matches) {
		case 1:
			if err := b.checkOwnerLocked(instanceID, matches[0].TabUUID, session); err != nil {
				return "", err
			}
			return matches[0].TabUUID, nil
		case 0:
			return "", &ResolutionError{Code: ErrCodeTabNotFound, Message: "no tab matches the target; list tabs with browser_list_tabs"}
		default:
			cands := make([]string, len(matches))
			for i, t := range matches {
				cands[i] = tabSummary(t)
			}
			return "", &ResolutionError{Code: ErrCodeTabAmbiguous, Message: "multiple tabs match the target; disambiguate with tab.uuid or a narrower pattern", Candidates: cands}
		}
	}
	switch ref.Role {
	case "active":
		for _, t := range tabs {
			if t.Active {
				if err := b.checkOwnerLocked(instanceID, t.TabUUID, session); err != nil {
					return "", err
				}
				return t.TabUUID, nil
			}
		}
		return "", &ResolutionError{Code: ErrCodeTabNotFound, Message: "no active tab is known for this browser; refresh the extension tab registry"}
	case "new":
		// The extension creates the tab; the bus assigns a fresh UUID that the
		// extension maps onto the new native tab id.
		tabUUID := newID("tab")
		return tabUUID, nil
	default:
		return "", &ResolutionError{Code: ErrCodeInvalidTarget, Message: "tab target must set uuid, url, title, or role (active|new)"}
	}
}

type tabKey struct {
	instanceID string
	tabUUID    string
}

func (b *Bus) ownerFor(instanceID, tabUUID string) string {
	if b.owners == nil {
		return ""
	}
	return b.owners[tabKey{instanceID, tabUUID}]
}

func (b *Bus) setOwner(instanceID, tabUUID, session string) {
	if b.owners == nil {
		b.owners = map[tabKey]string{}
	}
	b.owners[tabKey{instanceID, tabUUID}] = session
}

func (b *Bus) clearOwner(instanceID, tabUUID, session string) {
	if b.owners == nil {
		return
	}
	k := tabKey{instanceID, tabUUID}
	if b.owners[k] != session {
		return
	}
	delete(b.owners, k)
}

// checkOwnerLocked enforces one in-flight session per tab. Must be called with
// b.mu held.
func (b *Bus) checkOwnerLocked(instanceID, tabUUID, session string) error {
	current := b.ownerFor(instanceID, tabUUID)
	if current == "" || current == session {
		return nil
	}
	return &ResolutionError{Code: ErrCodeTabBusy, Message: fmt.Sprintf("tab is busy with session %q; wait or target another tab", current)}
}

func (b *Bus) enqueueLocked(cmd *Command) {
	b.commands[cmd.CommandID] = cmd
	b.perInstance = append(b.perInstance, cmd.CommandID)
	b.stats.total++
	// Bound retained history: keep the global command list capped, dropping
	// oldest terminal entries (queued/in-flight commands are never dropped).
	if len(b.perInstance) > MaxCommandsPerInstance*4 {
		drop := b.perInstance[:len(b.perInstance)-MaxCommandsPerInstance*4]
		for _, id := range drop {
			if c, ok := b.commands[id]; ok && c.Status != StatusQueued && c.Status != StatusSent {
				b.deleteCommandLocked(id)
			}
		}
		b.perInstance = b.perInstance[len(b.perInstance)-MaxCommandsPerInstance*4:]
	}
}

// deleteCommandLocked removes a command and releases its owner lock. Must be
// called with b.mu held. It is used by the history-trim path to ensure a
// forcibly removed command does not leave a tab locked forever.
func (b *Bus) deleteCommandLocked(commandID string) {
	if cmd, ok := b.commands[commandID]; ok {
		b.clearOwner(cmd.InstanceID, cmd.TabUUID, cmd.SessionID)
	}
	delete(b.commands, commandID)
}

// MarkSent records the extension's acknowledgement that a command is being
// processed.
func (b *Bus) MarkSent(commandID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	cmd, ok := b.commands[commandID]
	if !ok {
		return &ResolutionError{Code: ErrCodeCommandNotFound, Message: "command not found"}
	}
	now := b.nowTime()
	cmd.Status = StatusSent
	cmd.SentAt = &now
	return nil
}

// Result completes a command and wakes any in-process waiter.
func (b *Bus) Result(ctx context.Context, req ResultRequest) error {
	b.mu.Lock()
	cmd, ok := b.commands[req.CommandID]
	if !ok {
		b.mu.Unlock()
		return &ResolutionError{Code: ErrCodeCommandNotFound, Message: "command not found"}
	}
	if req.Status == "processing" {
		now := b.nowTime()
		if cmd.Status == StatusQueued {
			cmd.Status = StatusSent
			cmd.SentAt = &now
		}
		b.mu.Unlock()
		return nil
	}
	now := b.nowTime()
	cmd.FinishedAt = &now
	switch req.Status {
	case StatusSucceeded:
		if req.Result != nil {
			switch {
			case cmd.Action == ActionScreenshot && b.screenshotSink != nil:
				url, path := b.persistScreenshot(ctx, cmd, req.Result.Data)
				req.Result.Data = url
				req.Result.Path = path
			case cmd.Action == ActionPDF && b.pdfSink != nil:
				url, path := b.persistPdf(ctx, cmd, req.Result.Data)
				req.Result.Data = url
				req.Result.Path = path
			}
		}
		cmd.Status = StatusSucceeded
		cmd.Result = req.Result
		cmd.Error = ""
		b.stats.succeeded++
	case StatusFailed:
		cmd.Status = StatusFailed
		cmd.Error = req.Error
		b.stats.failed++
	default:
		cmd.Status = StatusFailed
		if req.Error != "" {
			cmd.Error = req.Error
		} else {
			cmd.Error = fmt.Sprintf("unexpected extension status %q", req.Status)
		}
		b.stats.failed++
	}
	b.clearOwner(cmd.InstanceID, cmd.TabUUID, cmd.SessionID)
	waiter := b.waiters[req.CommandID]
	delete(b.waiters, req.CommandID)
	b.mu.Unlock()

	if waiter != nil {
		select {
		case waiter <- cmd:
		case <-ctx.Done():
		}
		close(waiter)
	}
	return nil
}

// persistScreenshot decodes a screenshot data URL, stores the PNG via the
// installed sink, and returns the reference URL to keep in the result. On any
// failure the original data is returned unchanged so the result is never lost.
func (b *Bus) persistScreenshot(ctx context.Context, cmd *Command, data string) (string, string) {
	const prefix = "data:image/png;base64,"
	if cmd.Action != ActionScreenshot || !strings.HasPrefix(data, prefix) {
		return data, ""
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, prefix))
	if err != nil {
		return data, ""
	}
	ref, err := b.screenshotSink(ctx, cmd.CommandID, raw)
	if err != nil {
		return data, ""
	}
	return ref.URL, ref.Path
}

// persistPdf decodes a PDF data URL, stores it via the installed sink, and
// returns the reference URL to keep in the result. On any failure the original
// data is returned unchanged so the result is never lost.
func (b *Bus) persistPdf(ctx context.Context, cmd *Command, data string) (string, string) {
	const prefix = "data:application/pdf;base64,"
	if cmd.Action != ActionPDF || !strings.HasPrefix(data, prefix) {
		return data, ""
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, prefix))
	if err != nil {
		return data, ""
	}
	ref, err := b.pdfSink(ctx, cmd.CommandID, raw)
	if err != nil {
		return data, ""
	}
	return ref.URL, ref.Path
}

// GetCommand returns a command snapshot (used by CLI polling).
func (b *Bus) GetCommand(commandID string) (Command, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	cmd, ok := b.commands[commandID]
	if !ok {
		return Command{}, &ResolutionError{Code: ErrCodeCommandNotFound, Message: "command not found"}
	}
	return *cmd, nil
}

// WaitForResult blocks until the command finishes or the context times out.
// On timeout it marks the command timed_out and wakes the extension path.
func (b *Bus) WaitForResult(ctx context.Context, commandID string, timeout time.Duration) (Command, error) {
	ch := make(chan *Command, 1)
	b.mu.Lock()
	if cmd, ok := b.commands[commandID]; ok {
		switch cmd.Status {
		case StatusSucceeded, StatusFailed, StatusTimedOut:
			b.mu.Unlock()
			close(ch)
			return *cmd, nil
		}
		b.waiters[commandID] = ch
	} else {
		b.mu.Unlock()
		close(ch)
		return Command{}, &ResolutionError{Code: ErrCodeCommandNotFound, Message: "command not found"}
	}
	b.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		b.timeoutCommand(commandID)
		cmd, _ := b.GetCommand(commandID)
		return cmd, ctx.Err()
	case <-timer.C:
		b.timeoutCommand(commandID)
		cmd, _ := b.GetCommand(commandID)
		return cmd, fmt.Errorf("command timed out after %s", timeout)
	case cmd := <-ch:
		if cmd == nil {
			return Command{}, &ResolutionError{Code: ErrCodeCommandNotFound, Message: "command not found"}
		}
		return *cmd, nil
	}
}

func (b *Bus) timeoutCommand(commandID string) {
	b.mu.Lock()
	cmd, ok := b.commands[commandID]
	if ok && (cmd.Status == StatusQueued || cmd.Status == StatusSent) {
		now := b.nowTime()
		cmd.Status = StatusTimedOut
		cmd.Error = "command timed out"
		cmd.FinishedAt = &now
		b.clearOwner(cmd.InstanceID, cmd.TabUUID, cmd.SessionID)
		b.stats.timedOut++
	}
	// WaitForResult (the only channel reader) calls this path itself on
	// timeout and then fetches the state via GetCommand, so closing the
	// waiter is sufficient — there is nothing left to receive.
	if waiter := b.waiters[commandID]; ok && waiter != nil {
		delete(b.waiters, commandID)
		close(waiter)
	}
	b.mu.Unlock()
}

// Queue returns queued commands for an instance (the extension's SSE
// reconnect / alarm fallback path).
func (b *Bus) Queue(instanceID string) []Command {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Command, 0)
	for _, cmd := range b.commands {
		if cmd.InstanceID == instanceID && cmd.Status == StatusQueued {
			out = append(out, *cmd)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// ---------------------------------------------------------------------------
// SSE subscription
// ---------------------------------------------------------------------------

// Subscribe registers a channel for instanceID events. The returned cancel
// unsubscribes. A nil/empty instanceID receives no events; callers subscribe
// per instance.
func (b *Bus) Subscribe(instanceID string) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	b.mu.Lock()
	if b.subs[instanceID] == nil {
		b.subs[instanceID] = make(map[chan Event]struct{})
	}
	b.subs[instanceID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs[instanceID], ch)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, cancel
}

// push sends an event to an instance's subscribers without holding the bus
// mutex while sending on channels (which could block the caller).
func (b *Bus) pushLocked(instanceID string, ev Event) {
	b.mu.Lock()
	subs := make([]chan Event, 0, len(b.subs[instanceID]))
	for ch := range b.subs[instanceID] {
		subs = append(subs, ch)
	}
	b.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
			// Slow subscriber: drop the event; the alarm fallback re-polls the
			// queue, so nothing is permanently lost.
		}
	}
}

// ---------------------------------------------------------------------------
// Sweeper
// ---------------------------------------------------------------------------

func (b *Bus) sweep() {
	now := b.nowTime()
	b.mu.Lock()
	offline := make([]string, 0)
	for id, inst := range b.instances {
		if inst.Online && now.Sub(inst.LastSeenAt) > InstanceTTL {
			inst.Online = false
			offline = append(offline, id)
		}
	}
	// Fail queued commands for offline instances.
	for _, id := range offline {
		for _, cmd := range b.commands {
			if cmd.InstanceID == id && cmd.Status == StatusQueued {
				cmd.Status = StatusFailed
				cmd.Error = "browser_offline"
				now2 := b.nowTime()
				cmd.FinishedAt = &now2
				b.clearOwner(cmd.InstanceID, cmd.TabUUID, cmd.SessionID)
				b.stats.failed++
				if w := b.waiters[cmd.CommandID]; w != nil {
					delete(b.waiters, cmd.CommandID)
					close(w)
				}
			}
		}
	}
	b.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func validAction(action string) bool {
	switch action {
	case ActionNavigate, ActionClick, ActionType, ActionPress, ActionScroll,
		ActionWait, ActionScrape, ActionEval, ActionScreenshot, ActionNewTab, ActionCloseTab,
		ActionFocus, ActionSelect, ActionCheck, ActionHover, ActionDrag, ActionBringToFront,
		ActionGetCookies, ActionSetCookie, ActionPDF,
		ActionStorage,
		ActionOrchestrate:
		return true
	}
	return false
}

// globMatch is a small glob matcher supporting '*' and '?'; it treats the
// pattern as a substring-style glob against the value (case-insensitive).
func globMatch(pattern, value string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	value = strings.ToLower(strings.TrimSpace(value))
	if pattern == "" {
		return true
	}
	if pattern == "*" {
		return value != ""
	}
	return globMatchSeq(pattern, value)
}

func globMatchSeq(p, s string) bool {
	for len(p) > 0 {
		switch p[0] {
		case '*':
			for i := 0; i <= len(s); i++ {
				if globMatchSeq(p[1:], s[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(s) == 0 {
				return false
			}
			p, s = p[1:], s[1:]
		default:
			if len(s) == 0 || s[0] != p[0] {
				return false
			}
			p, s = p[1:], s[1:]
		}
	}
	return len(s) == 0
}

func instanceSummary(inst Instance) string {
	label := inst.Label
	if label == "" {
		label = inst.Browser
	}
	return fmt.Sprintf("%s (instance_id=%s, %s)", label, inst.InstanceID, inst.Browser)
}

func tabSummary(t Tab) string {
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = t.URL
	}
	return fmt.Sprintf("%s (uuid=%s, url=%s)", title, t.TabUUID, t.URL)
}

// ErrIsResolution reports whether err is a typed ResolutionError.
func ErrIsResolution(err error) bool {
	var re *ResolutionError
	return errors.As(err, &re)
}

// ResolutionCode extracts the stable code from a ResolutionError.
func ResolutionCode(err error) string {
	var re *ResolutionError
	if errors.As(err, &re) {
		return re.Code
	}
	return ""
}
