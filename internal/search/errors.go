package search

import (
	"fmt"
	"time"
)

type ErrorCode string

const (
	ErrRateLimited      ErrorCode = "RATE_LIMITED"
	ErrQuotaExceeded    ErrorCode = "QUOTA_EXCEEDED"
	ErrInvalidAPIKey    ErrorCode = "INVALID_API_KEY"
	ErrNoResults        ErrorCode = "NO_RESULTS"
	ErrNetworkError     ErrorCode = "NETWORK_ERROR"
	ErrTimeout          ErrorCode = "TIMEOUT"
	ErrInvalidQuery     ErrorCode = "INVALID_QUERY"
	ErrProviderDown     ErrorCode = "PROVIDER_DOWN"
	ErrResponseTooLarge ErrorCode = "RESPONSE_TOO_LARGE"
	ErrUnknown          ErrorCode = "UNKNOWN"
)

type SearchError struct {
	Provider   string
	Code       ErrorCode
	Message    string
	RetryAfter time.Duration
}

func (e *SearchError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s][%s] %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
func (e *SearchError) Is(target error) bool {
	t, ok := target.(*SearchError)
	return ok && e.Code == t.Code
}
func (e *SearchError) Retryable() bool {
	return e.Code == ErrRateLimited || e.Code == ErrNetworkError || e.Code == ErrTimeout || e.Code == ErrProviderDown
}

func statusError(provider string, status int, message string) *SearchError {
	code := ErrUnknown
	switch {
	case status == 429:
		code = ErrRateLimited
	case status == 401:
		code = ErrInvalidAPIKey
	case status == 403:
		code = ErrQuotaExceeded
	case status >= 500:
		code = ErrProviderDown
	case status == 400:
		code = ErrInvalidQuery
	}
	return &SearchError{Provider: provider, Code: code, Message: fmt.Sprintf("HTTP %d: %s", status, message)}
}
