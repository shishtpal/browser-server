package images

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// List returns the most recent gallery entries, newest first.
func (s *Service) List(ctx context.Context, limit int) ([]Image, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,prompt,provider,model,image_size,content_type,filename,size_bytes,created_at FROM ai_images ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Image{}
	for rows.Next() {
		var x Image
		var t string
		if err := rows.Scan(&x.ID, &x.Prompt, &x.Provider, &x.Model, &x.ImageSize, &x.ContentType, &x.Filename, &x.SizeBytes, &t); err != nil {
			return nil, err
		}
		x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
		out = append(out, x)
	}
	return out, rows.Err()
}

// Get returns one gallery entry by ID.
func (s *Service) Get(ctx context.Context, id string) (Image, error) {
	var x Image
	var t string
	err := s.db.QueryRowContext(ctx, `SELECT id,prompt,provider,model,image_size,content_type,filename,size_bytes,created_at FROM ai_images WHERE id=?`, id).Scan(&x.ID, &x.Prompt, &x.Provider, &x.Model, &x.ImageSize, &x.ContentType, &x.Filename, &x.SizeBytes, &t)
	x.CreatedAt, _ = time.Parse(time.RFC3339Nano, t)
	return x, err
}

// Delete removes a gallery entry and its backing image file.
func (s *Service) Delete(ctx context.Context, id string) error {
	x, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM ai_images WHERE id=?`, id); err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(s.root, x.Filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Read returns the gallery entry together with the raw image bytes.
func (s *Service) Read(ctx context.Context, id string) (Image, []byte, error) {
	x, err := s.Get(ctx, id)
	if err != nil {
		return x, nil, err
	}
	b, err := os.ReadFile(filepath.Join(s.root, x.Filename))
	return x, b, err
}
