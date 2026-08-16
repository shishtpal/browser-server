package videos

import "testing"

func TestValidateProviderParams(t *testing.T) {
	specs := []ParamSpec{
		{Key: "width", Type: "number", Min: float64Ptr(64), Max: float64Ptr(1920)},
		{Key: "mode", Type: "select", Options: []string{"ti2vid", "keyframes"}},
		{Key: "seed", Type: "number"},
	}
	valid := map[string]any{
		"width": float64(1152),
		"mode":  "keyframes",
	}
	if err := validateProviderParams(specs, valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	tooWide := map[string]any{"width": float64(9999)}
	if err := validateProviderParams(specs, tooWide); err == nil {
		t.Fatal("width above max should be rejected")
	}
	badMode := map[string]any{"mode": "burn"}
	if err := validateProviderParams(specs, badMode); err == nil {
		t.Fatal("unknown select option should be rejected")
	}
	textSeed := map[string]any{"seed": "not-a-number"}
	if err := validateProviderParams(specs, textSeed); err == nil {
		t.Fatal("non-numeric seed should be rejected")
	}
	// Strings with trailing garbage must not partially parse (fmt.Sscanf would
	// happily accept "6abc" as 6).
	suffixSeed := map[string]any{"seed": "123abc"}
	if err := validateProviderParams(specs, suffixSeed); err == nil {
		t.Fatal("seed with trailing garbage should be rejected")
	}
	// Numeric strings with surrounding whitespace are legitimate.
	spaceSeed := map[string]any{"seed": " 42 "}
	if err := validateProviderParams(specs, spaceSeed); err != nil {
		t.Fatalf("numeric string seed rejected: %v", err)
	}
}

func TestValidateAgnesConstraints(t *testing.T) {
	if err := validateAgnesConstraints(map[string]any{"num_frames": float64(121)}); err != nil {
		t.Fatalf("valid frame count rejected: %v", err)
	}
	if err := validateAgnesConstraints(map[string]any{"num_frames": float64(120)}); err == nil {
		t.Fatal("frame count not satisfying 8n+1 should be rejected")
	}
	if err := validateAgnesConstraints(map[string]any{"frame_rate": float64(0.5)}); err == nil {
		t.Fatal("frame rate below 1 should be rejected")
	}
	one := []any{"https://a.png"}
	if err := validateAgnesConstraints(map[string]any{"mode": "keyframes", "extra_body.image": one}); err == nil {
		t.Fatal("keyframes with a single image should be rejected")
	}
	two := []any{"https://a.png", "https://b.png"}
	if err := validateAgnesConstraints(map[string]any{"mode": "keyframes", "extra_body.image": two}); err != nil {
		t.Fatalf("keyframes with two images rejected: %v", err)
	}
	// A single image without keyframes mode is ordinary image-to-video.
	if err := validateAgnesConstraints(map[string]any{"image": one}); err != nil {
		t.Fatalf("image-to-video with one image rejected: %v", err)
	}
	// Keyframe URLs with no explicit mode are promoted to keyframes in place,
	// since Agnes requires mode=keyframes alongside extra_body.image.
	promoted := map[string]any{"extra_body.image": two}
	if err := validateAgnesConstraints(promoted); err != nil {
		t.Fatalf("keyframe promotion rejected: %v", err)
	}
	if promoted["mode"] != "keyframes" {
		t.Fatalf("mode = %v, want keyframes", promoted["mode"])
	}
	// num_frames is clamped against the configured spec bounds when present.
	specs := []ParamSpec{{Key: "num_frames", Type: "number", Min: float64Ptr(9), Max: float64Ptr(241)}}
	if err := validateAgnesConstraintsWithSpecs(map[string]any{"num_frames": float64(441)}, specs); err == nil {
		t.Fatal("num_frames above spec max should be rejected")
	}
	if err := validateAgnesConstraintsWithSpecs(map[string]any{"num_frames": float64(241)}, specs); err != nil {
		t.Fatalf("num_frames at spec max rejected: %v", err)
	}
	// The documented absolute ceiling applies even when the spec allows more.
	wide := []ParamSpec{{Key: "num_frames", Type: "number", Min: float64Ptr(9), Max: float64Ptr(500)}}
	if err := validateAgnesConstraintsWithSpecs(map[string]any{"num_frames": float64(697)}, wide); err == nil {
		t.Fatal("num_frames above the hard ceiling should be rejected")
	}
}

func float64Ptr(v float64) *float64 { return &v }
