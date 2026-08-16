package videos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// VideoStatus enumerates the async generation lifecycle.
type VideoStatus string

const (
	StatusQueued    VideoStatus = "queued"
	StatusProgress  VideoStatus = "in_progress"
	StatusCompleted VideoStatus = "completed"
	StatusFailed    VideoStatus = "failed"
)

// Video is one persisted gallery entry (a generation task + its result).
type Video struct {
	ID           string      `json:"id"`
	TaskID       string      `json:"task_id,omitempty"`
	Prompt       string      `json:"prompt"`
	Provider     string      `json:"provider"`
	Model        string      `json:"model"`
	Params       string      `json:"params"` // JSON-encoded request parameters
	ContentType  string      `json:"content_type"`
	Filename     string      `json:"filename"`
	SizeBytes    int64       `json:"size_bytes"`
	Status       VideoStatus `json:"status"`
	Progress     int         `json:"progress"`
	VideoURL     string      `json:"video_url,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	Duration     float64     `json:"seconds,omitempty"`
	Size         string      `json:"size,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
}

const columns = "id,task_id,prompt,provider,model,params,content_type,filename,size_bytes,status,progress,video_url,error_message,seconds,size,created_at,completed_at"

func scanVideo(s interface {
	Scan(...any) error
}) (Video, error) {
	var v Video
	var taskID, params, contentType, filename, videoURL, errorMsg, size, t string
	var sizeBytes int64
	var progress int
	var completedAt sql.NullString
	if err := s.Scan(&v.ID, &taskID, &v.Prompt, &v.Provider, &v.Model, &params, &contentType, &filename, &sizeBytes, &v.Status, &progress, &videoURL, &errorMsg, &v.Duration, &size, &t, &completedAt); err != nil {
		return v, err
	}
	v.TaskID = taskID
	v.Params = params
	v.ContentType = contentType
	v.Filename = filename
	v.SizeBytes = sizeBytes
	v.Progress = progress
	v.VideoURL = videoURL
	v.ErrorMessage = errorMsg
	v.Size = size
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
	if completedAt.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, completedAt.String)
		if err == nil {
			v.CompletedAt = &parsed
		}
	}
	return v, nil
}

// List returns the most recent gallery entries, newest first.
func (s *Service) List(ctx context.Context, limit int) ([]Video, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+columns+` FROM ai_videos ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Get returns one gallery entry by ID.
func (s *Service) Get(ctx context.Context, id string) (Video, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM ai_videos WHERE id=?`, id)
	return scanVideo(row)
}

// Pending returns tasks still awaiting or undergoing generation.
func (s *Service) Pending(ctx context.Context) ([]Video, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+columns+` FROM ai_videos WHERE status IN (?, ?) ORDER BY created_at ASC`, StatusQueued, StatusProgress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// InsertTask persists a freshly created generation task with status queued.
func (s *Service) InsertTask(ctx context.Context, v Video) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ai_videos (id,task_id,prompt,provider,model,params,content_type,filename,size_bytes,status,progress,video_url,error_message,seconds,size,created_at,completed_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.TaskID, v.Prompt, v.Provider, v.Model, v.Params, v.ContentType, v.Filename, v.SizeBytes,
		v.Status, v.Progress, v.VideoURL, v.ErrorMessage, v.Duration, v.Size,
		v.CreatedAt.Format(time.RFC3339Nano), nilTime(v.CompletedAt))
	return err
}

// UpdateStatus advances a task's lifecycle state and, on completion, records
// the backing file metadata.
func (s *Service) UpdateStatus(ctx context.Context, id, taskID string, status VideoStatus, progress int, videoURL, errorMsg string, sizeBytes int64, filename, contentType string, duration float64, size string, completedAt *time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE ai_videos SET status=?, progress=?, task_id=?, video_url=?, error_message=?, size_bytes=?, filename=?, content_type=?, seconds=?, size=?, completed_at=? WHERE id=?`,
		status, progress, taskID, videoURL, errorMsg, sizeBytes, filename, contentType, duration, size, nilTime(completedAt), id)
	return err
}

// Delete removes a gallery entry and its backing video file.
func (s *Service) Delete(ctx context.Context, id string) error {
	v, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM ai_videos WHERE id=?`, id); err != nil {
		return err
	}
	if v.Filename != "" {
		if err = os.Remove(filepath.Join(s.root, v.Filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Read returns the gallery entry together with the raw video bytes.
func (s *Service) Read(ctx context.Context, id string) (Video, []byte, error) {
	v, err := s.Get(ctx, id)
	if err != nil {
		return v, nil, err
	}
	if v.Filename == "" {
		return v, nil, errors.New("video file not ready")
	}
	b, err := os.ReadFile(filepath.Join(s.root, v.Filename))
	return v, b, err
}

// FilePath returns the on-disk location of a gallery video.
func (s *Service) FilePath(filename string) string {
	return filepath.Join(s.root, filename)
}

func nilTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339Nano)
}

// paramJSON marshals request parameters for storage.
func paramJSON(params map[string]any) string {
	if len(params) == 0 {
		return "{}"
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func parseParamJSON(raw string) map[string]any {
	out := map[string]any{}
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
