package helpers

import "testing"

func TestValidatorRequired(t *testing.T) {
	v := NewValidator()
	v.Required("title", "  ")
	v.Required("name", "ok")
	if v.OK() {
		t.Fatal("expected validation to fail for blank title")
	}
	if _, ok := v.Fields()["title"]; !ok {
		t.Errorf("expected title error, got %v", v.Fields())
	}
	if _, ok := v.Fields()["name"]; ok {
		t.Errorf("did not expect name error, got %v", v.Fields())
	}
}

func TestValidatorPositiveID(t *testing.T) {
	v := NewValidator()
	v.PositiveID("user_id", 0)
	v.PositiveID("other_id", 5)
	if _, ok := v.Fields()["user_id"]; !ok {
		t.Errorf("expected user_id error for 0, got %v", v.Fields())
	}
	if _, ok := v.Fields()["other_id"]; ok {
		t.Errorf("did not expect other_id error for 5, got %v", v.Fields())
	}
}

func TestValidatorURL(t *testing.T) {
	cases := map[string]bool{
		"https://example.com":     true,
		"http://localhost:8080/x": true,
		"ftp://example.com":       false,
		"example.com":             false,
		"":                        false,
		"https://":                false,
	}
	for input, wantOK := range cases {
		v := NewValidator()
		v.URL("url", input)
		if v.OK() != wantOK {
			t.Errorf("URL(%q): got OK=%v, want %v (fields=%v)", input, v.OK(), wantOK, v.Fields())
		}
	}
}

func TestURLHostname(t *testing.T) {
	cases := map[string]string{
		"https://Example.com/path":                "example.com",
		"https://user:pass@example.com:8443/path": "example.com",
		"https://example.com?section=history":     "example.com",
		"http://[::1]:8080/path":                  "::1",
		"chrome://history":                        "",
		"not a URL":                               "",
	}
	for input, want := range cases {
		if got := URLHostname(input); got != want {
			t.Errorf("URLHostname(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidatorEmail(t *testing.T) {
	cases := map[string]bool{
		"john@example.com": true,
		"not-an-email":     false,
		"":                 false,
		"a@b.co":           true,
	}
	for input, wantOK := range cases {
		v := NewValidator()
		v.Email("email", input)
		if v.OK() != wantOK {
			t.Errorf("Email(%q): got OK=%v, want %v (fields=%v)", input, v.OK(), wantOK, v.Fields())
		}
	}
}

func TestValidatorFirstErrorWins(t *testing.T) {
	v := NewValidator()
	v.Required("url", "")
	v.URL("url", "")
	if got := v.Fields()["url"]; got != "is required" {
		t.Errorf("expected first error to be preserved, got %q", got)
	}
}
