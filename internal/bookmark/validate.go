package bookmark

import (
	"fmt"
	"strings"
)

func ValidateTitle(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("title must not be empty")
	}
	if len(value) > MaxTitleLength {
		return "", fmt.Errorf("title must be %d characters or fewer", MaxTitleLength)
	}
	return value, nil
}

func ValidateDescription(value string) error {
	if len(value) > MaxDescriptionLength {
		return fmt.Errorf("description must be %d characters or fewer", MaxDescriptionLength)
	}
	return nil
}

func ValidateURL(value string) error {
	if len(value) > MaxURLLength {
		return fmt.Errorf("url must be %d characters or fewer", MaxURLLength)
	}
	return nil
}

func ValidateFolderPath(value string) error {
	if len(value) > MaxFolderPathLength {
		return fmt.Errorf("folder_path must be %d characters or fewer", MaxFolderPathLength)
	}
	return nil
}

func ValidateTags(tags []string) error {
	for i, tag := range tags {
		if len(tag) > MaxTagLength {
			return fmt.Errorf("tags[%d] must be %d characters or fewer", i, MaxTagLength)
		}
	}
	return nil
}

func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			out = append(out, tag)
		}
	}
	return out
}
