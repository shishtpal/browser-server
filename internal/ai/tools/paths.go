package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"browser-server/internal/ai/config"
)

// resolveBinary finds the full path to a binary using the configured paths.
// Resolution order: explicit binaries map → additional_dirs → system PATH.
// Returns the bare name (last resort) when nothing is found, so the caller
// can still attempt exec and get a standard "not found" error.
func resolveBinary(name string, paths config.PathsConfig) string {
	// 1. Explicit override
	if paths.Binaries != nil {
		if full, ok := paths.Binaries[name]; ok && full != "" {
			return full
		}
		// Also accept config keys without a platform executable extension, so
		// e.g. resolveBinary("uv.exe", ...) matches a "uv" override on Windows.
		if base := stripExecExt(name); base != name {
			if full, ok := paths.Binaries[base]; ok && full != "" {
				return full
			}
		}
	}
	// 2. Additional dirs (searched in order)
	for _, dir := range paths.AdditionalDirs {
		if candidate := findExecutable(name, dir); candidate != "" {
			return candidate
		}
	}
	// 3. System PATH
	if resolved, err := exec.LookPath(name); err == nil {
		return resolved
	}
	// Last resort: return the bare name so exec gives a standard error.
	return name
}

// childEnv returns os.Environ() with PATH modified to prepend directories from
// both configured path sources, in precedence order:
//
//  1. parent directories of explicit paths.binaries entries (keys sorted so
//     precedence is deterministic across server starts), then
//  2. paths.additional_dirs in their configured order.
//
// Both precede the inherited PATH. Directories are cleaned with filepath.Clean
// and deduplicated preserving first occurrence (case-insensitively on
// Windows). Returns nil (inherit parent env) when neither source is
// configured, so callers can leave cmd.Env unset for the default behavior.
func childEnv(paths config.PathsConfig) []string {
	dirs := childPathDirs(paths)
	if len(dirs) == 0 {
		return nil
	}
	env := os.Environ()
	currentPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	newPath := strings.Join(dirs, sep)
	if currentPath != "" {
		newPath += sep + currentPath
	}
	for i, e := range env {
		// PATH may appear as "Path=" on Windows; compare case-insensitively.
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = "PATH=" + newPath
			return env
		}
	}
	return append(env, "PATH="+newPath)
}

// childPathDirs returns the directories to prepend to a child process PATH:
// the parent directories of paths.binaries entries (sorted by binary key, so
// precedence never depends on map iteration order), followed by
// paths.additional_dirs in their configured order. Directories are cleaned
// with filepath.Clean and deduplicated preserving first occurrence;
// comparison is case-insensitive on Windows.
func childPathDirs(paths config.PathsConfig) []string {
	var dirs []string
	seen := make(map[string]bool)
	addDir := func(dir string) {
		dir = filepath.Clean(dir)
		if dir == "" || dir == "." {
			return
		}
		key := dir
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			return
		}
		seen[key] = true
		dirs = append(dirs, dir)
	}

	if paths.Binaries != nil {
		keys := make([]string, 0, len(paths.Binaries))
		for k := range paths.Binaries {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if p := paths.Binaries[k]; p != "" {
				addDir(filepath.Dir(p))
			}
		}
	}
	for _, dir := range paths.AdditionalDirs {
		addDir(dir)
	}
	return dirs
}

// stripExecExt returns name without a Windows executable extension
// (.exe, .bat, .cmd, .com). It returns name unchanged when there is no match.
func stripExecExt(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range []string{".exe", ".bat", ".cmd", ".com"} {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

// findExecutable searches for an executable file named `name` (or name.exe on
// Windows) in `dir`. Returns the full path if found, empty string otherwise.
func findExecutable(name, dir string) string {
	candidates := []string{filepath.Join(dir, name)}
	if runtime.GOOS == "windows" {
		candidates = nil
		for _, ext := range []string{".exe", ".bat", ".cmd", ""} {
			candidates = append(candidates, filepath.Join(dir, name+ext))
		}
	}
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err != nil || info.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o100 == 0 {
			continue
		}
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		return abs
	}
	return ""
}
