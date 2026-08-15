package browser

import (
	"encoding/json"
	"testing"

	corebrowser "browser-server/internal/browser"
)

func TestValidateStorage(t *testing.T) {
	spec := specs[corebrowser.ActionStorage]
	valid := []args{
		{StorageType: "local", StorageAction: "get", Key: "k"},
		{StorageType: "session", StorageAction: "get", Key: "k"},
		{StorageType: "local", StorageAction: "set", Key: "k", Value: "v"},
		{StorageType: "local", StorageAction: "remove", Key: "k"},
		{StorageType: "local", StorageAction: "list"},
		{StorageType: "local", StorageAction: "list", Raw: true},
	}
	for _, a := range valid {
		if err := spec.validate(&a); err != nil {
			t.Errorf("args %+v should be valid: %v", a, err)
		}
	}
	invalid := []args{
		{StorageType: "cookie", StorageAction: "get", Key: "k"},
		{StorageType: "", StorageAction: "get", Key: "k"},
		{StorageType: "local", StorageAction: "count"},
		{StorageType: "local", StorageAction: "", Key: "k"},
		{StorageType: "local", StorageAction: "get"},
		{StorageType: "local", StorageAction: "get", Key: "  "},
		{StorageType: "local", StorageAction: "set", Key: "k"},
		{StorageType: "local", StorageAction: "remove"},
	}
	for _, a := range invalid {
		if err := spec.validate(&a); err == nil {
			t.Errorf("args %+v should be invalid", a)
		}
	}
}

func TestStorageSpecParams(t *testing.T) {
	spec := specs[corebrowser.ActionStorage]
	params := spec.params(&args{
		StorageType: "local", StorageAction: "set",
		Key: "k", Value: "v", Raw: true,
	})
	if params["type"] != "local" || params["action"] != "set" {
		t.Fatalf("type/action missing: %+v", params)
	}
	if params["key"] != "k" || params["value"] != "v" {
		t.Fatalf("key/value missing: %+v", params)
	}
	if params["raw"] != true {
		t.Fatalf("raw missing: %+v", params)
	}
}

func TestValidateStepsAcceptsStorageSteps(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionNavigate, Params: jsonRaw(`{"url":"https://example.com"}`)},
		{Action: corebrowser.ActionStorage, Params: jsonRaw(`{"type":"local","action":"set","key":"token","value":"abc"}`)},
		{Action: corebrowser.ActionStorage, Params: jsonRaw(`{"type":"session","action":"get","key":"token","raw":true}`)},
		{Action: corebrowser.ActionStorage, Params: jsonRaw(`{"type":"local","action":"list"}`)},
	}
	if err := validateSteps(steps); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateStepsRejectsBadStorageStep(t *testing.T) {
	steps := []step{
		{Action: corebrowser.ActionStorage, Params: jsonRaw(`{"type":"local","action":"remove"}`)},
	}
	if err := validateSteps(steps); err == nil {
		t.Fatal("expected storage step without key to error")
	}
}

func TestStorageToolSchema(t *testing.T) {
	b := corebrowser.New()
	defer b.Close()
	tool := toolByName(Tools(&corebrowser.LocalClient{Bus: b}), "browser_storage")
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema, &schema); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("no properties: %+v", schema)
	}
	for _, k := range []string{"type", "action", "key", "value", "raw"} {
		if _, ok := props[k]; !ok {
			t.Errorf("schema missing %q", k)
		}
	}
}
