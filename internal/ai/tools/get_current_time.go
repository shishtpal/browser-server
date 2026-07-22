package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func registerGetCurrentTime(r *Registry) {
	r.add(Tool{
		Name:        "get_current_time",
		Category:    "General",
		Description: "Get the current server time",
		Schema:      json.RawMessage(`{"type":"object","properties":{"timezone":{"type":"string"}},"additionalProperties":false}`),
		Execute:     currentTime,
	})
}

func currentTime(_ context.Context, raw json.RawMessage) (any, error) {
	var a struct {
		Timezone string `json:"timezone"`
	}
	if err := strict(raw, &a, map[string]bool{"timezone": true}); err != nil {
		return nil, err
	}
	loc := time.UTC
	if a.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(a.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone")
		}
	}
	return map[string]string{
		"time":     time.Now().In(loc).Format(time.RFC3339),
		"timezone": loc.String(),
	}, nil
}
