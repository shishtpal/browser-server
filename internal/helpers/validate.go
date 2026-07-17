package helpers

import (
	"net/mail"
	"net/url"
	"strings"
)

// Validator accumulates field-level validation errors keyed by JSON field name.
//
//	v := helpers.NewValidator()
//	v.Required("title", todo.Title)
//	if !v.OK() {
//		helpers.WriteValidationError(w, v.Fields())
//		return
//	}
type Validator struct {
	fields map[string]string
}

func NewValidator() *Validator {
	return &Validator{fields: map[string]string{}}
}

// OK reports whether no validation errors have been recorded.
func (v *Validator) OK() bool { return len(v.fields) == 0 }

// Fields returns the accumulated field errors.
func (v *Validator) Fields() map[string]string { return v.fields }

// add records the first error seen for a field (later errors don't overwrite).
func (v *Validator) add(field, message string) {
	if _, exists := v.fields[field]; !exists {
		v.fields[field] = message
	}
}

// Required fails if the trimmed string value is empty.
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.add(field, "is required")
	}
}

// PositiveID fails if the integer id is not greater than zero.
func (v *Validator) PositiveID(field string, value int) {
	if value <= 0 {
		v.add(field, "must be a positive integer")
	}
}

// URL fails if the value is empty or is not a valid absolute http(s) URL.
func (v *Validator) URL(field, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		v.add(field, "is required")
		return
	}
	if !validURL(value) {
		v.add(field, "must be a valid http(s) URL")
	}
}

// Email fails if the value is empty or is not a valid email address.
func (v *Validator) Email(field, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		v.add(field, "is required")
		return
	}
	if _, err := mail.ParseAddress(value); err != nil {
		v.add(field, "must be a valid email address")
	}
}

func validURL(value string) bool {
	u, err := url.Parse(value)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

// URLHostname returns a normalized hostname for an HTTP(S) URL. Ports and
// credentials are intentionally excluded so all variants group as one domain.
func URLHostname(value string) string {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
