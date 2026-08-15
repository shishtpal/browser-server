package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	browserconfig "browser-server/internal/ai/browser/config"
	aiconfig "browser-server/internal/ai/config"
	"browser-server/internal/ai/images"
	aimcp "browser-server/internal/ai/mcp"
	"browser-server/internal/ai/tts"
	"browser-server/internal/ai/voice"
	"browser-server/internal/quiz"
	quizconfig "browser-server/internal/quiz/config"

	"github.com/gorilla/mux"
)

const (
	keepSecretSentinel = "__KEEP__"
	maxConfigFileBytes = 4 << 20
)

type configFileMeta struct {
	Name   string `json:"name"`
	Class  string `json:"class"`
	Reload string `json:"reload"`
}

var (
	configNamePattern = regexp.MustCompile(`^bs-[a-z-]+\.json$`)
	secretKeyPattern  = regexp.MustCompile(`(?i)(api[_-]?key|secret|token|authorization|password|credential|private[_-]?key)`)
	errConfigTooLarge = errors.New("config file exceeds the 4 MiB limit")
	configFiles       = map[string]configFileMeta{
		"bs-ai-config.json":       {Name: "bs-ai-config.json", Class: "core", Reload: "restart_required"},
		"bs-ai-models.json":       {Name: "bs-ai-models.json", Class: "core", Reload: "restart_required"},
		"bs-ai-mcp.json":          {Name: "bs-ai-mcp.json", Class: "core", Reload: "restart_required"},
		"bs-ai-tts.json":          {Name: "bs-ai-tts.json", Class: "leaf", Reload: "hot_reload"},
		"bs-ai-image-models.json": {Name: "bs-ai-image-models.json", Class: "leaf", Reload: "hot_reload"},
		"bs-ai-voice.json":        {Name: "bs-ai-voice.json", Class: "leaf", Reload: "hot_reload"},
		"bs-quiz-config.json":     {Name: "bs-quiz-config.json", Class: "leaf", Reload: "hot_reload"},
		"bs-browser-config.json":  {Name: "bs-browser-config.json", Class: "leaf", Reload: "hot_reload"},
	}
)

type configFileListItem struct {
	configFileMeta
	Exists  bool       `json:"exists"`
	Size    int64      `json:"size"`
	ModTime *time.Time `json:"modified_at,omitempty"`
}

type configFileResponse struct {
	configFileMeta
	Content string `json:"content"`
}

type configWriteRequest struct {
	Content string `json:"content"`
}

// RegisterAdmin adds config-file routes to an administrator-only subrouter.
func (m *Module) RegisterAdmin(router *mux.Router) {
	router.HandleFunc("/config/files", m.listConfigFiles).Methods(http.MethodGet)
	router.HandleFunc("/config/files/{name}", m.getConfigFile).Methods(http.MethodGet)
	router.HandleFunc("/config/files/{name}", m.putConfigFile).Methods(http.MethodPut)
	router.HandleFunc("/config/reload/{name}", m.reloadConfigFile).Methods(http.MethodPost)
}

func configMeta(name string) (configFileMeta, int, bool) {
	if !configNamePattern.MatchString(name) || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return configFileMeta{}, http.StatusBadRequest, false
	}
	meta, ok := configFiles[name]
	if !ok {
		return configFileMeta{}, http.StatusNotFound, false
	}
	return meta, 0, true
}

func (m *Module) configPath(name string) string {
	return filepath.Join(m.configDir, name)
}

func (m *Module) listConfigFiles(w http.ResponseWriter, _ *http.Request) {
	m.configMu.RLock()
	defer m.configMu.RUnlock()
	items := make([]configFileListItem, 0, len(configFiles))
	for _, meta := range configFiles {
		item := configFileListItem{configFileMeta: meta}
		info, err := os.Lstat(m.configPath(meta.Name))
		switch {
		case err == nil:
			if !info.Mode().IsRegular() {
				writeError(w, http.StatusInternalServerError, "config_path_invalid", meta.Name+" is not a regular file.")
				return
			}
			item.Exists = true
			item.Size = info.Size()
			modified := info.ModTime().UTC()
			item.ModTime = &modified
		case errors.Is(err, os.ErrNotExist):
			// Missing optional files remain visible so an administrator can
			// create them from the editor.
		default:
			writeError(w, http.StatusInternalServerError, "config_stat_failed", "Unable to inspect "+meta.Name+".")
			return
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	writeJSON(w, http.StatusOK, items)
}

func (m *Module) getConfigFile(w http.ResponseWriter, r *http.Request) {
	m.configMu.RLock()
	defer m.configMu.RUnlock()
	name := mux.Vars(r)["name"]
	meta, status, ok := configMeta(name)
	if !ok {
		writeError(w, status, "invalid_config_name", "Config file is not in the administrator whitelist.")
		return
	}
	content, err := readBoundedFile(m.configPath(name))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "config_not_found", "Config file does not exist.")
		return
	}
	if errors.Is(err, errConfigTooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "config_too_large", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_read_failed", "Unable to read config file.")
		return
	}
	redacted, err := redactJSON(content)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_config", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configFileResponse{configFileMeta: meta, Content: string(redacted)})
}

func (m *Module) putConfigFile(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	meta, status, ok := configMeta(name)
	if !ok {
		writeError(w, status, "invalid_config_name", "Config file is not in the administrator whitelist.")
		return
	}
	var request configWriteRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxConfigFileBytes))
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must contain a config content string.")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Request body must contain one JSON value.")
		return
	}

	m.configMu.Lock()
	defer m.configMu.Unlock()

	path := m.configPath(name)
	candidate, err := restoreMaskedJSON([]byte(request.Content), path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "masked_secret_restore_failed", err.Error())
		return
	}
	if err := m.validateConfigFile(name, candidate); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "config_validation_failed", err.Error())
		return
	}
	if err := writeAtomic(path, candidate); err != nil {
		writeError(w, http.StatusInternalServerError, "config_write_failed", err.Error())
		return
	}

	response := map[string]any{"saved": true, "reload": meta.Reload}
	if meta.Class == "leaf" {
		warning, reloadErr := m.reloadLeaf(name)
		if reloadErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"saved": true, "reload": "failed", "error": reloadErr.Error(),
			})
			return
		}
		response["reload"] = "hot_reloaded"
		if warning != "" {
			response["warning"] = warning
			response["restart_required"] = true
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (m *Module) reloadConfigFile(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	meta, status, ok := configMeta(name)
	if !ok {
		writeError(w, status, "invalid_config_name", "Config file is not in the administrator whitelist.")
		return
	}
	if meta.Class != "leaf" {
		writeError(w, http.StatusNotFound, "reload_not_supported", "Core config files require a server restart.")
		return
	}

	m.configMu.Lock()
	defer m.configMu.Unlock()
	warning, err := m.reloadLeaf(name)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "config_reload_failed", err.Error())
		return
	}
	response := map[string]any{"reloaded": true, "reload": "hot_reloaded"}
	if warning != "" {
		response["warning"] = warning
		response["restart_required"] = true
	}
	writeJSON(w, http.StatusOK, response)
}

func redactJSON(content []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, fmt.Errorf("config contains invalid JSON: %w", err)
	}
	redactValue(value)
	redacted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode redacted config: %w", err)
	}
	return append(redacted, '\n'), nil
}

func redactValue(value any) {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if text, ok := child.(string); ok && secretKeyPattern.MatchString(key) && !strings.HasPrefix(text, "env:") {
				current[key] = keepSecretSentinel
				continue
			}
			redactValue(child)
		}
	case []any:
		for _, child := range current {
			redactValue(child)
		}
	}
}

func restoreMaskedJSON(candidate []byte, path string) ([]byte, error) {
	var edited any
	if err := json.Unmarshal(candidate, &edited); err != nil {
		return nil, fmt.Errorf("config contains invalid JSON: %w", err)
	}
	var existing any
	if containsKeepSentinel(edited) {
		onDisk, err := readBoundedFile(path)
		if err == nil {
			if err := json.Unmarshal(onDisk, &existing); err != nil {
				return nil, fmt.Errorf("existing config contains invalid JSON and cannot restore masked secrets: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read existing config: %w", err)
		}
	}

	restored, err := restoreKeepValue(edited, existing, "$", existing != nil)
	if err != nil {
		return nil, err
	}
	formatted, err := json.MarshalIndent(restored, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return append(formatted, '\n'), nil
}

func containsKeepSentinel(value any) bool {
	switch current := value.(type) {
	case string:
		return current == keepSecretSentinel
	case map[string]any:
		for _, child := range current {
			if containsKeepSentinel(child) {
				return true
			}
		}
	case []any:
		for _, child := range current {
			if containsKeepSentinel(child) {
				return true
			}
		}
	}
	return false
}

func restoreKeepValue(edited, existing any, path string, existingPresent bool) (any, error) {
	if text, ok := edited.(string); ok && text == keepSecretSentinel {
		if !existingPresent {
			return nil, fmt.Errorf("%s uses %s but no existing value is available", path, keepSecretSentinel)
		}
		return existing, nil
	}

	switch current := edited.(type) {
	case map[string]any:
		oldObject, _ := existing.(map[string]any)
		for key, child := range current {
			oldChild, present := oldObject[key]
			restored, err := restoreKeepValue(child, oldChild, path+"."+key, present)
			if err != nil {
				return nil, err
			}
			current[key] = restored
		}
	case []any:
		oldArray, _ := existing.([]any)
		for index, child := range current {
			var oldChild any
			present := index < len(oldArray)
			if present {
				oldChild = oldArray[index]
			}
			restored, err := restoreKeepValue(child, oldChild, fmt.Sprintf("%s[%d]", path, index), present)
			if err != nil {
				return nil, err
			}
			current[index] = restored
		}
	}
	return edited, nil
}

func (m *Module) validateConfigFile(name string, content []byte) error {
	// Every file gets a syntax check before its package-specific semantics.
	var syntax any
	if err := json.Unmarshal(content, &syntax); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	switch name {
	case "bs-ai-tts.json":
		return tts.ValidateBytes(content)
	case "bs-ai-image-models.json":
		return images.ValidateBytes(content)
	case "bs-ai-voice.json":
		return voice.ValidateBytes(content)
	case "bs-quiz-config.json":
		return quizconfig.ValidateBytes(content)
	case "bs-browser-config.json":
		return browserconfig.ValidateBytes(content)
	case "bs-ai-mcp.json":
		return aimcp.ValidateBytes(content, m.configPath(name))
	case "bs-ai-config.json":
		models, err := readBoundedFile(m.configPath("bs-ai-models.json"))
		if errors.Is(err, os.ErrNotExist) {
			return nil // startup treats a missing catalog as disabled
		}
		if err != nil {
			return fmt.Errorf("read bs-ai-models.json: %w", err)
		}
		return aiconfig.ValidateBytes(content, models, m.configDir)
	case "bs-ai-models.json":
		mainConfig, err := readBoundedFile(m.configPath("bs-ai-config.json"))
		if errors.Is(err, os.ErrNotExist) {
			return nil // the catalog is dormant until the main config exists
		}
		if err != nil {
			return fmt.Errorf("read bs-ai-config.json: %w", err)
		}
		return aiconfig.ValidateBytes(mainConfig, content, m.configDir)
	default:
		return errors.New("unsupported config file")
	}
}

func (m *Module) reloadLeaf(name string) (string, error) {
	path := m.configPath(name)
	dataDir := filepath.Join(m.configDir, ".data")
	if m.cfg != nil {
		dataDir = m.cfg.ResolvePath(".data")
	}

	switch name {
	case "bs-ai-tts.json":
		config, err := tts.LoadPath(path)
		if err != nil {
			return "", err
		}
		service, err := tts.New(config, dataDir)
		if err != nil {
			return "", fmt.Errorf("initialize TTS service: %w", err)
		}
		if m.holders == nil || m.holders.TTS == nil {
			if service != nil {
				_ = service.Close()
			}
			return "", errors.New("TTS service holder is unavailable")
		}
		if err := m.holders.TTS.Swap(service, func(old *tts.Service) error { return old.Close() }); err != nil {
			log.Printf("close retired TTS service: %v", err)
		}
		return "", nil

	case "bs-ai-image-models.json":
		config, err := images.LoadPath(path)
		if err != nil {
			return "", err
		}
		service, err := images.New(config, dataDir)
		if err != nil {
			return "", fmt.Errorf("initialize image service: %w", err)
		}
		if m.holders == nil || m.holders.Images == nil {
			if service != nil {
				_ = service.Close()
			}
			return "", errors.New("image service holder is unavailable")
		}
		if err := m.holders.Images.Swap(service, func(old *images.Service) error { return old.Close() }); err != nil {
			log.Printf("close retired image service: %v", err)
		}
		return "", nil

	case "bs-ai-voice.json":
		config, err := voice.LoadPath(path)
		if err != nil {
			return "", err
		}
		m.voice.Store(config)
		return "", nil

	case "bs-quiz-config.json":
		config, restartRequired, err := quizconfig.LoadPathPreservingRuntime(path, quizconfig.Get())
		if err != nil {
			return "", err
		}
		quiz.SetDefaultScheduler(config.Scheduler)
		if restartRequired {
			return "Quiz rules were hot-reloaded, but boot-time settings remain unchanged in this process and require a server restart.", nil
		}
		return "", nil

	case "bs-browser-config.json":
		if _, err := browserconfig.LoadPath(path); err != nil {
			return "", err
		}
		return "", nil
	}
	return "", errors.New("config file does not support hot reload")
}

func readBoundedFile(path string) ([]byte, error) {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !pathInfo.Mode().IsRegular() {
		return nil, errors.New("config path is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("config path is not a regular file")
	}
	if info.Size() > maxConfigFileBytes {
		return nil, errConfigTooLarge
	}
	content, err := io.ReadAll(io.LimitReader(file, maxConfigFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxConfigFileBytes {
		return nil, errConfigTooLarge
	}
	return content, nil
}

func writeAtomic(path string, content []byte) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("config path is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect config target: %w", err)
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}

	backupPath := fmt.Sprintf("%s.bak.%d.%d", path, os.Getpid(), time.Now().UnixNano())
	hadTarget := true
	if err := os.Rename(path, backupPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			hadTarget = false
		} else {
			return fmt.Errorf("prepare config replacement: %w", err)
		}
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		if hadTarget {
			if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
				return fmt.Errorf("commit config: %v (rollback failed: %v)", err, restoreErr)
			}
		}
		return fmt.Errorf("commit config: %w", err)
	}
	committed = true
	if hadTarget {
		if err := os.Remove(backupPath); err != nil {
			log.Printf("remove config backup %s: %v", backupPath, err)
		}
	}
	return nil
}
