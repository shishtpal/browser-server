// Package quizconfig loads bs-quiz-config.json from the executable directory
// (or BS_QUIZ_CONFIG_PATH). When the file is missing or enabled is false the
// quiz feature is a no-op: no DB init, no routes, no AI tools.
package quizconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/quiz"
)

const defaultConfigFile = "bs-quiz-config.json"

// LimitsConfig bounds user-supplied payloads.
type LimitsConfig struct {
	MaxQuestionLength     int   `json:"max_question_length"`
	MaxExplanationLength  int   `json:"max_explanation_length"`
	MaxOptionLength       int   `json:"max_option_length"`
	MaxChronologyItems    int   `json:"max_chronology_items"`
	MaxOptionsPerQuestion int   `json:"max_options_per_question"`
	MaxImageBytes         int64 `json:"max_image_bytes"`
	MaxPaperSize          int   `json:"max_paper_size"`
	MaxPapersPerUser      int   `json:"max_papers_per_user"`
}

// AIToolsConfig gates the quiz AI tools independently of the feature toggle.
type AIToolsConfig struct {
	Enabled         bool `json:"enabled"`
	SearchQuestions bool `json:"search_questions"`
	ManageQuestion  bool `json:"manage_question"`
}

// PaperGenConfig controls paper generation behaviour.
type PaperGenConfig struct {
	DefaultSampleStrategy              string `json:"default_sample_strategy"`
	AllowDuplicateQuestionsWithinPaper bool   `json:"allow_duplicate_questions_within_paper"`
}

// Config is the parsed bs-quiz-config.json.
type Config struct {
	Enabled              bool           `json:"enabled"`
	Path                 string         `json:"-"`
	DBPath               string         `json:"db_path"`
	ImageDir             string         `json:"image_dir"`
	Limits               LimitsConfig   `json:"limits"`
	AllowedQuestionTypes []string       `json:"allowed_question_types"`
	AllowedDifficulties  []string       `json:"allowed_difficulties"`
	TagCategories        []string       `json:"tag_categories"`
	AITools              AIToolsConfig  `json:"ai_tools"`
	PaperGeneration      PaperGenConfig `json:"paper_generation"`
	RetentionDays        int            `json:"retention_days"`
	CORSEnabled          bool           `json:"cors_enabled"`
	// Scheduler picks the spaced-repetition engine: "sm2" (default) or "fsrs".
	Scheduler string `json:"scheduler"`
}

var global atomic.Pointer[Config]

func init() {
	global.Store(&Config{})
}

// Get returns the last-loaded config. It is never nil; before Load it returns
// a disabled zero config.
func Get() *Config { return global.Load() }

// Rules converts the configured limits and vocabularies into the validation
// policy the quiz domain package enforces. This is what makes the `limits`,
// `allowed_question_types`, and `allowed_difficulties` sections of
// bs-quiz-config.json actually take effect. Unset sections fall back to the
// domain defaults so a zero config still validates sensibly.
func (c *Config) Rules() quiz.Rules {
	rules := quiz.DefaultRules()
	if c == nil {
		return rules
	}
	if c.Limits.MaxQuestionLength > 0 {
		rules.Limits.MaxQuestionLength = c.Limits.MaxQuestionLength
	}
	if c.Limits.MaxExplanationLength > 0 {
		rules.Limits.MaxExplanationLength = c.Limits.MaxExplanationLength
	}
	if c.Limits.MaxOptionLength > 0 {
		rules.Limits.MaxOptionLength = c.Limits.MaxOptionLength
	}
	if c.Limits.MaxChronologyItems > 0 {
		rules.Limits.MaxChronologyItems = c.Limits.MaxChronologyItems
	}
	if c.Limits.MaxOptionsPerQuestion > 0 {
		rules.Limits.MaxOptionsPerQuestion = c.Limits.MaxOptionsPerQuestion
	}
	if c.Limits.MaxPaperSize > 0 {
		rules.Limits.MaxPaperSize = c.Limits.MaxPaperSize
	}
	if len(c.AllowedQuestionTypes) > 0 {
		rules.AllowedTypes = c.AllowedQuestionTypes
	}
	if len(c.AllowedDifficulties) > 0 {
		rules.AllowedDifficulties = c.AllowedDifficulties
	}
	return rules
}

// ResolveImageDir returns the absolute image directory. A relative
// `image_dir` is joined onto dataPath; an absolute one is used as-is. Both
// the server's startup mkdir and the image handlers go through this so they
// can never disagree about where images live.
func (c *Config) ResolveImageDir(dataPath string) string {
	if filepath.IsAbs(c.ImageDir) {
		return c.ImageDir
	}
	return filepath.Join(dataPath, c.ImageDir)
}

// ResetForTest replaces the global config with a fresh disabled one. It is
// only intended for tests; production callers should use Load.
func ResetForTest() { global.Store(&Config{}) }

// SetForTest installs cfg as the active global config. Test-only helper.
func SetForTest(cfg *Config) { global.Store(cfg) }

// DefaultLimitsForTest returns a LimitsConfig populated with the same
// defaults Load applies. Test-only helper.
func DefaultLimitsForTest() LimitsConfig {
	return defaultConfig().Limits
}

// DefaultPaperGenerationForTest returns a PaperGenConfig populated with the
// same defaults Load applies. Test-only helper.
func DefaultPaperGenerationForTest() PaperGenConfig {
	return defaultConfig().PaperGeneration
}

// Load reads bs-quiz-config.json, applies defaults, validates, and atomically
// replaces the global config. A missing file yields a disabled config and is
// not an error (mirrors bs-ai-config.json behaviour).
func Load() (*Config, error) {
	path := os.Getenv("BS_QUIZ_CONFIG_PATH")
	if path == "" {
		exeDir, err := aiconfig.ExecutableDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(exeDir, defaultConfigFile)
	}
	return LoadPath(path)
}

// LoadPath reloads an explicit whitelisted path and atomically publishes the
// resulting rules. It intentionally ignores BS_QUIZ_CONFIG_PATH so the admin
// file API always reloads the same file it just committed.
func LoadPath(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			cfg.Enabled = false
			cfg.Path = path
			global.Store(cfg)
			log.Printf("Quiz disabled: no config file at %s", path)
			return cfg, nil
		}
		return nil, fmt.Errorf("read quiz config: %w", err)
	}

	cfg, err := parseBytes(content, path)
	if err != nil {
		return nil, err
	}
	global.Store(cfg)
	return cfg, nil
}

// LoadPathPreservingRuntime reloads quiz rules while retaining fields whose
// resources are wired only at boot. Publishing those changes early could
// expose handlers without a database or redirect image operations away from
// the initialized data location.
func LoadPathPreservingRuntime(path string, current *Config) (*Config, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read quiz config: %w", err)
	}
	cfg, err := parseBytes(content, path)
	if err != nil {
		return nil, false, err
	}
	if current == nil {
		current = &Config{}
	}
	restartRequired := cfg.Enabled != current.Enabled || cfg.DBPath != current.DBPath ||
		cfg.ImageDir != current.ImageDir || cfg.CORSEnabled != current.CORSEnabled
	cfg.Enabled = current.Enabled
	cfg.DBPath = current.DBPath
	cfg.ImageDir = current.ImageDir
	cfg.CORSEnabled = current.CORSEnabled
	global.Store(cfg)
	return cfg, restartRequired, nil
}

// ValidateBytes performs the same defaulting and semantic checks as Load
// without changing the process-global active quiz rules.
func ValidateBytes(content []byte) error {
	_, err := parseBytes(content, defaultConfigFile)
	return err
}

func parseBytes(content []byte, path string) (*Config, error) {
	cfg := defaultConfig()
	cfg.Enabled = true
	cfg.Path = path
	if err := json.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse quiz config: %w", err)
	}
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("quiz config: %w", err)
	}
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		DBPath:   ".data/quiz.db",
		ImageDir: ".data/quiz-images",
		Limits: LimitsConfig{
			MaxQuestionLength:     2000,
			MaxExplanationLength:  20000,
			MaxOptionLength:       500,
			MaxChronologyItems:    20,
			MaxOptionsPerQuestion: 10,
			MaxImageBytes:         5 * 1024 * 1024,
			MaxPaperSize:          200,
			MaxPapersPerUser:      100,
		},
		AllowedQuestionTypes: []string{"single_choice", "multiple_choice", "input", "chronology"},
		AllowedDifficulties:  []string{"easy", "medium", "hard"},
		TagCategories:        []string{"subject", "topic", "sub_topic"},
		AITools:              AIToolsConfig{Enabled: true, SearchQuestions: true, ManageQuestion: true},
		PaperGeneration: PaperGenConfig{
			DefaultSampleStrategy:              "random",
			AllowDuplicateQuestionsWithinPaper: false,
		},
		RetentionDays: 365,
		Scheduler:     "sm2",
	}
}

func validate(cfg *Config) error {
	l := cfg.Limits
	if l.MaxQuestionLength < 1 || l.MaxQuestionLength > 100000 {
		return fmt.Errorf("limits.max_question_length must be between 1 and 100000")
	}
	if l.MaxExplanationLength < 1 || l.MaxExplanationLength > 1000000 {
		return fmt.Errorf("limits.max_explanation_length must be between 1 and 1000000")
	}
	if l.MaxOptionLength < 1 || l.MaxOptionLength > 10000 {
		return fmt.Errorf("limits.max_option_length must be between 1 and 10000")
	}
	if l.MaxChronologyItems < 2 || l.MaxChronologyItems > 100 {
		return fmt.Errorf("limits.max_chronology_items must be between 2 and 100")
	}
	if l.MaxOptionsPerQuestion < 2 || l.MaxOptionsPerQuestion > 50 {
		return fmt.Errorf("limits.max_options_per_question must be between 2 and 50")
	}
	if l.MaxImageBytes < 1024 || l.MaxImageBytes > 50*1024*1024 {
		return fmt.Errorf("limits.max_image_bytes must be between 1024 and 52428800")
	}
	if l.MaxPaperSize < 1 || l.MaxPaperSize > 1000 {
		return fmt.Errorf("limits.max_paper_size must be between 1 and 1000")
	}
	if l.MaxPapersPerUser < 1 || l.MaxPapersPerUser > 10000 {
		return fmt.Errorf("limits.max_papers_per_user must be between 1 and 10000")
	}
	if len(cfg.AllowedQuestionTypes) == 0 {
		return fmt.Errorf("allowed_question_types must not be empty")
	}
	if len(cfg.AllowedDifficulties) == 0 {
		return fmt.Errorf("allowed_difficulties must not be empty")
	}
	if len(cfg.TagCategories) == 0 {
		return fmt.Errorf("tag_categories must not be empty")
	}
	cfg.Scheduler = strings.ToLower(strings.TrimSpace(cfg.Scheduler))
	if cfg.Scheduler == "" {
		cfg.Scheduler = quiz.SchedulerSM2
	}
	if cfg.Scheduler != quiz.SchedulerSM2 && cfg.Scheduler != quiz.SchedulerFSRS {
		return fmt.Errorf(`scheduler must be %q or %q`, quiz.SchedulerSM2, quiz.SchedulerFSRS)
	}
	return nil
}
