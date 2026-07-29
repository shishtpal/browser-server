// Package prompt holds the prompt and prompt-folder domain logic shared by the
// REST API handlers in internal/handlers and the AI tools in
// internal/ai/tools.
//
// Both entry points previously repeated the same SELECT column lists, folder
// join, ownership checks, and length limits. They now share this package so
// the prompt model is defined in one place. HTTP and AI-tool concerns stay in
// the callers; see view.go for the two response shapes.
package prompt

import (
	"fmt"
	"strings"
)

// Field limits shared by the API handlers and the AI tools.
const (
	MaxTitleLength       = 200
	MaxContentLength     = 10000
	MaxDescriptionLength = 2000
	MaxTagLength         = 100
)

// ValidateTitle trims and length-checks a required prompt title.
func ValidateTitle(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("title must not be empty")
	}
	if len(trimmed) > MaxTitleLength {
		return "", fmt.Errorf("title must be %d characters or fewer", MaxTitleLength)
	}
	return trimmed, nil
}

// ValidateContent length-checks prompt body text.
func ValidateContent(value string) error {
	if len(value) > MaxContentLength {
		return fmt.Errorf("content must be %d characters or fewer", MaxContentLength)
	}
	return nil
}

// ValidateDescription length-checks an optional description.
func ValidateDescription(value string) error {
	if len(value) > MaxDescriptionLength {
		return fmt.Errorf("description must be %d characters or fewer", MaxDescriptionLength)
	}
	return nil
}

// ValidateTags length-checks each tag, reporting the offending index.
func ValidateTags(tags []string) error {
	for i, tag := range tags {
		if len(tag) > MaxTagLength {
			return fmt.Errorf("tags[%d] must be %d characters or fewer", i, MaxTagLength)
		}
	}
	return nil
}
