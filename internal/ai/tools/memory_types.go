package tools

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

var memoryIDPattern = regexp.MustCompile(`^memory_[a-z0-9_-]{1,80}$`)

type memoryMeta struct {
	ID            string     `json:"id"`
	Timestamp     time.Time  `json:"timestamp"`
	UpdatedAt     time.Time  `json:"updated_at,omitempty"`
	Title         string     `json:"title,omitempty"`
	Type          string     `json:"type"`
	TargetID      string     `json:"target_id,omitempty"`
	Relationship  string     `json:"relationship,omitempty"`
	References    []string   `json:"references,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	Category      string     `json:"category,omitempty"`
	Importance    string     `json:"importance,omitempty"`
	Source        string     `json:"source"`
	Lazy          bool       `json:"lazy,omitempty"`
	LazyTrigger   string     `json:"lazy_trigger,omitempty"`
	LazyExpiresAt *time.Time `json:"lazy_expires_at,omitempty"`
}

type memoryData struct {
	Metadata memoryMeta `json:"metadata"`
	Content  string     `json:"content,omitempty"`
}

type scoredMemory struct {
	data  memoryData
	score float64
}

func validID(id string) error {
	if !memoryIDPattern.MatchString(id) {
		return fmt.Errorf("invalid memory id")
	}
	return nil
}

func newMemoryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "memory_" + hex.EncodeToString(b), nil
}
