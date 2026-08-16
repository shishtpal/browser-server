package videos

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GenerateRequest is the normalized generation request handed to the service.
type GenerateRequest struct {
	Prompt   string         `json:"prompt"`
	Provider string         `json:"provider,omitempty"`
	Model    string         `json:"model,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
}

// Validate checks the request against the selected model's parameter schema and
// any provider-specific constraints (e.g. Agnes frame rules).
func (r GenerateRequest) Validate(models []Model) error {
	if r.Prompt == "" {
		return errors.New("prompt is required")
	}
	var model Model
	for _, m := range models {
		if m.ID == r.Model {
			model = m
		}
	}
	if model.ID == "" {
		return errors.New("unknown video model")
	}
	return validateProviderParams(model.Parameters, r.Params)
}

func validateProviderParams(specs []ParamSpec, params map[string]any) error {
	for _, spec := range specs {
		// "prompt" is a top-level request field, not a generic parameter, so it
		// is validated separately (GenerateRequest.Validate) rather than here.
		if spec.Key == "prompt" {
			continue
		}
		raw, present := params[spec.Key]
		if !present || isEmpty(raw) {
			if spec.Required {
				return fmt.Errorf("parameter %q (%s) is required", spec.Key, spec.Label)
			}
			continue
		}
		switch spec.Type {
		case "number":
			f, ok := toFloat(raw)
			if !ok {
				return fmt.Errorf("parameter %q must be a number", spec.Key)
			}
			if spec.Min != nil && f < *spec.Min {
				return fmt.Errorf("parameter %q must be >= %v", spec.Key, *spec.Min)
			}
			if spec.Max != nil && f > *spec.Max {
				return fmt.Errorf("parameter %q must be <= %v", spec.Key, *spec.Max)
			}
		case "select":
			s, ok := raw.(string)
			if !ok {
				return fmt.Errorf("parameter %q must be a string", spec.Key)
			}
			if !contains(spec.Options, s) {
				return fmt.Errorf("parameter %q must be one of %v", spec.Key, spec.Options)
			}
		case "image_urls":
			if _, ok := raw.([]any); !ok {
				return fmt.Errorf("parameter %q must be a list of URLs", spec.Key)
			}
		}
	}
	return nil
}

// validateAgnesConstraints enforces Agnes-specific numeric rules that cannot
// be expressed purely through min/max/step schema bounds.
//
// Agnes's documented absolute ceiling for num_frames is 441 and any value must
// satisfy the 8n+1 rule — both are enforced here. The keyframes requirement
// and the num_frames minimum are validated against the configured model spec
// when one is supplied (the config is the source of truth), with the hardcoded
// documented minimum as a fallback.
func validateAgnesConstraints(params map[string]any) error {
	return validateAgnesConstraintsWithSpecs(params, nil)
}

func validateAgnesConstraintsWithSpecs(params map[string]any, specs []ParamSpec) error {
	spec := findParamSpec(specs, "num_frames")
	if v, ok := params["num_frames"]; ok && !isEmpty(v) {
		f, ok := toFloat(v)
		if !ok {
			return errors.New("num_frames must be a number")
		}
		n := int(math.Round(f))
		min := 9 // documented absolute minimum
		if spec != nil && spec.Min != nil {
			min = int(math.Round(*spec.Min))
		}
		max := 441 // documented absolute ceiling; spec bounds are checked too
		if spec != nil && spec.Max != nil && *spec.Max < float64(max) {
			max = int(math.Round(*spec.Max))
		}
		if n < min || n > max {
			return fmt.Errorf("num_frames must be between %d and %d", min, max)
		}
		if (n-1)%8 != 0 {
			return errors.New("num_frames must follow the 8n+1 rule")
		}
	}
	if v, ok := params["frame_rate"]; ok {
		f, ok := toFloat(v)
		if !ok {
			return errors.New("frame_rate must be a number")
		}
		if f < 1 || f > 60 {
			return errors.New("frame_rate must be between 1 and 60")
		}
	}
	// Keyframes are a sequence of frames the model interpolates between, so
	// Agnes requires at least two images. Users often supply the images and
	// leave mode unset or on ti2vid; supplying keyframe URLs promotes the
	// request to keyframes mode in place (createPayload reads the normalized
	// params), because Agnes requires mode=keyframes together with
	// extra_body.image. Requesting the mode with zero or one image is rejected.
	mode, _ := params["mode"].(string)
	imgs := stringSlice(params["extra_body.image"])
	if len(imgs) > 0 && mode != "keyframes" {
		params["mode"] = "keyframes"
		mode = "keyframes"
	}
	if mode == "keyframes" && len(imgs) < 2 {
		return errors.New("keyframes require at least two image URLs")
	}
	return nil
}

func findParamSpec(specs []ParamSpec, key string) *ParamSpec {
	for i := range specs {
		if specs[i].Key == key {
			return &specs[i]
		}
	}
	return nil
}

// validateOpenRouterConstraints enforces rules that cannot be expressed purely
// through schema bounds: OpenRouter only distinguishes a first and a last
// frame, so frame_images accepts at most two image URLs.
func validateOpenRouterConstraints(params map[string]any) error {
	if imgs := stringSlice(params["frame_images"]); len(imgs) > 2 {
		return errors.New("frame_images accepts at most two image URLs (first and last frame)")
	}
	return nil
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case []string:
		return len(x) == 0
	}
	return false
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// toInt coerces a JSON number or numeric string to an integer. Select params
// surface as strings ("6"), so this normalizes them for OpenRouter's typed
// request fields.
func toInt(v any) (int, bool) {
	f, ok := toFloat(v)
	if !ok {
		return 0, false
	}
	n := int(math.Round(f))
	if math.Abs(f-float64(n)) > 1e-6 {
		return 0, false
	}
	return n, true
}

func toBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case float64:
		return x != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return false, false
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	}
	if s, ok := v.(string); ok {
		// Reject trailing garbage ("6abc"), which fmt.Sscanf would accept.
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}
