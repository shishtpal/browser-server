package videos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"browser-server/internal/ai/store"
)

// Service owns the gallery database, the HTTP client used to reach providers,
// and the background poller that advances async generation tasks.
type Service struct {
	cfg    Config
	db     *sql.DB
	root   string
	client *http.Client
	stop   chan struct{}
	kick   chan struct{}
	wg     sync.WaitGroup
}

// New opens (or creates) the gallery database under dataDir and starts the
// poller. A nil service is returned when the feature is disabled. The
// AI_VIDEO_DB_PATH / AI_VIDEO_VIDEO_DIR environment variables take precedence
// over the config values, and relative paths resolve against dataDir so the
// feature honors DATA_PATH and stays consistent with a hot reload.
func New(cfg Config, dataDir string) (*Service, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	dbPath := cfg.DBPath
	if v := os.Getenv("AI_VIDEO_DB_PATH"); v != "" {
		dbPath = v
	}
	if dbPath == "" {
		dbPath = filepath.Join(dataDir, "ai-videos.db")
	} else if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(dataDir, dbPath)
	}
	root := cfg.VideoDir
	if v := os.Getenv("AI_VIDEO_VIDEO_DIR"); v != "" {
		root = v
	}
	if root == "" {
		root = filepath.Join(dataDir, "ai-videos")
	} else if !filepath.IsAbs(root) {
		root = filepath.Join(dataDir, root)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:"+filepath.ToSlash(dbPath)+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE IF NOT EXISTS ai_videos (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL DEFAULT '',
		prompt TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		params TEXT NOT NULL,
		content_type TEXT NOT NULL DEFAULT '',
		filename TEXT NOT NULL DEFAULT '',
		size_bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		progress INTEGER NOT NULL DEFAULT 0,
		video_url TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		seconds REAL NOT NULL DEFAULT 0,
		size TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		completed_at TEXT
	)`); err != nil {
		db.Close()
		return nil, err
	}
	// Hand each provider the OpenRouter attribution values injected from
	// bs-ai-config.json so its requests can carry HTTP-Referer / X-Title.
	for name, p := range cfg.Providers {
		p.OpenRouterSiteURL = cfg.OpenRouterSiteURL
		p.OpenRouterAppName = cfg.OpenRouterAppName
		cfg.Providers[name] = p
	}
	s := &Service{
		cfg:    cfg,
		db:     db,
		root:   root,
		client: &http.Client{},
		stop:   make(chan struct{}),
		kick:   make(chan struct{}, 1),
	}
	s.wg.Add(1)
	go s.pollLoop()
	return s, nil
}

// Close stops the poller and releases the gallery database. It waits for an
// in-flight advance — including its result download — which can occupy the
// full provider timeout for large videos on a hot reload. Safe on nil.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	close(s.stop)
	s.wg.Wait()
	return s.db.Close()
}

// Config returns the configuration the service was constructed from.
func (s *Service) Config() Config { return s.cfg }

func (s *Service) resolve(r GenerateRequest) (string, Provider, Model, error) {
	pn := r.Provider
	if pn == "" {
		pn = s.cfg.DefaultProvider
	}
	p, ok := s.cfg.Providers[pn]
	if !ok {
		return "", Provider{}, Model{}, errors.New("unknown video provider")
	}
	m := r.Model
	var mc Model
	if m == "" {
		mc = p.Models[0]
		for _, v := range p.Models {
			if v.Default {
				mc = v
				break
			}
		}
		m = mc.ID
	} else {
		for _, v := range p.Models {
			if v.ID == m {
				mc = v
				break
			}
		}
	}
	if mc.ID == "" {
		return "", Provider{}, Model{}, errors.New("unknown video model")
	}
	return pn, p, mc, nil
}

// Submit creates a provider task and persists a queued gallery record. The
// background poller picks the task up to advance it to completion.
func (s *Service) Submit(ctx context.Context, r GenerateRequest) (Video, error) {
	if r.Prompt == "" {
		return Video{}, errors.New("prompt is required")
	}
	pn, p, mc, err := s.resolve(r)
	if err != nil {
		return Video{}, err
	}
	if err := validateProviderParams(mc.Parameters, r.Params); err != nil {
		return Video{}, err
	}
	impl, err := newProviderImpl(p.Type)
	if err != nil {
		return Video{}, err
	}
	videoID, err := impl.Create(ctx, p, mc, r)
	if err != nil {
		if errors.Is(err, ErrProvider) {
			return Video{}, err
		}
		return Video{}, fmt.Errorf("%w", err)
	}
	v := Video{
		ID:        store.NewID("vid"),
		TaskID:    videoID,
		Prompt:    r.Prompt,
		Provider:  pn,
		Model:     mc.ID,
		Params:    paramJSON(r.Params),
		Status:    StatusQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.InsertTask(ctx, v); err != nil {
		return Video{}, err
	}
	select {
	case s.kick <- struct{}{}:
	default:
	}
	return v, nil
}

func (s *Service) pollLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	// Kick once shortly after startup to catch tasks created before the poller
	// started (e.g. on a hot reload).
	time.AfterFunc(500*time.Millisecond, func() {
		select {
		case s.kick <- struct{}{}:
		default:
		}
	})
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.pollOnce()
		case <-s.kick:
			s.pollOnce()
		}
	}
}

// pollOnce polls every non-terminal record. Each advance runs synchronously
// on the poller goroutine and is tracked in s.wg, so Close waits for any
// in-flight advance (including its result download) before the DB closes and
// a hot reload never lets a retired service write to a closed store. Advances
// are sequential by design: two advances of the same task cannot overlap
// within one service instance, keeping at-least-once result delivery without
// extra locking.
func (s *Service) pollOnce() {
	pending, err := s.Pending(context.Background())
	if err != nil {
		return
	}
	for _, v := range pending {
		// Skip records retired while an earlier advance in this batch ran.
		select {
		case <-s.stop:
			return
		default:
		}
		s.wg.Add(1)
		func() {
			defer s.wg.Done()
			s.advance(v)
		}()
	}
}

func (s *Service) advance(v Video) {
	// Bail immediately if the service is shutting down (hot reload).
	select {
	case <-s.stop:
		return
	default:
	}
	p, ok := s.cfg.Providers[v.Provider]
	if !ok {
		_ = s.UpdateStatus(context.Background(), v.ID, v.TaskID, StatusFailed, v.Progress, "", "provider configuration missing", 0, "", "", 0, "", nil)
		return
	}
	impl, err := newProviderImpl(p.Type)
	if err != nil {
		_ = s.UpdateStatus(context.Background(), v.ID, v.TaskID, StatusFailed, v.Progress, "", err.Error(), 0, "", "", 0, "", nil)
		return
	}
	// Give each task its own budget: the poll and, once completed, the video
	// download must both fit inside a single context. Video files can be large,
	// so this is the provider timeout (minutes) rather than the shared poll tick.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.RequestTimeoutSeconds)*time.Second)
	defer cancel()
	res, err := impl.Poll(ctx, p, v.TaskID, v.Model)
	if err != nil {
		// A permanent upstream failure (the provider reported "failed", a
		// completed result had no URL, or the task is gone) is recorded as
		// failed rather than retried forever. Transient errors leave the record
		// untouched so the next poll attempt can recover it.
		if res.Status == StatusFailed {
			_ = s.UpdateStatus(ctx, v.ID, v.TaskID, StatusFailed, res.Progress, "", err.Error(), 0, "", "", 0, "", nil)
			return
		}
		log.Printf("video poll %s: %v", v.ID, err)
		return
	}
	if res.Status != StatusCompleted {
		_ = s.UpdateStatus(ctx, v.ID, v.TaskID, res.Status, res.Progress, "", "", 0, "", "", 0, "", nil)
		return
	}
	data, contentType, derr := s.fetchResult(ctx, impl, p, res)
	if derr != nil {
		_ = s.UpdateStatus(ctx, v.ID, v.TaskID, StatusFailed, 100, res.VideoURL, derr.Error(), 0, "", "", 0, "", nil)
		return
	}
	filename := v.ID + ".mp4"
	if err := os.WriteFile(filepath.Join(s.root, filename), data, 0600); err != nil {
		_ = s.UpdateStatus(ctx, v.ID, v.TaskID, StatusFailed, 100, res.VideoURL, err.Error(), 0, "", "", 0, "", nil)
		return
	}
	// Some providers (OpenRouter) do not report duration/size in the job
	// response; fall back to the values the request asked for, which were
	// persisted in the record's params.
	seconds, size := res.Seconds, res.Size
	if seconds <= 0 || size == "" {
		if pj := parseParamJSON(v.Params); len(pj) > 0 {
			if seconds <= 0 {
				if d, ok := pj["duration"]; ok {
					if f, ok := toFloat(d); ok {
						seconds = f
					}
				}
			}
			if size == "" {
				if s, ok := pj["size"].(string); ok {
					size = s
				}
			}
		}
	}
	now := time.Now().UTC()
	_ = s.UpdateStatus(ctx, v.ID, v.TaskID, StatusCompleted, 100, res.VideoURL, "", int64(len(data)), filename, contentType, seconds, size, &now)
}

// fetchResult retrieves the completed video bytes. Providers that require an
// authenticated download implement contentFetcher; everyone else falls back to
// a plain GET of res.VideoURL.
func (s *Service) fetchResult(ctx context.Context, impl providerImpl, p Provider, res pollResult) ([]byte, string, error) {
	if fetcher, ok := impl.(contentFetcher); ok {
		return fetcher.Fetch(ctx, p, res)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, res.VideoURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrProvider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("%w: download status %d", ErrProvider, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 500<<20))
	if err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	return data, contentType, nil
}
