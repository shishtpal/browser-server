package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// childEnv returns os.Environ() with PATH modified to prepend additional_dirs.
// Returns nil (inherit parent env) when no additional dirs are configured, so
// callers can leave cmd.Env unset for the default behavior.
func childEnv(paths config.PathsConfig) []string {
	if len(paths.AdditionalDirs) == 0 {
		return nil
	}
	env := os.Environ()
	currentPath := os.Getenv("PATH")
	sep := string(os.PathListSeparator)
	newPath := strings.Join(paths.AdditionalDirs, sep)
	if currentPath != "" {
		newPath += sep + currentPath
	}
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = "PATH=" + newPath
			return env
		}
	}
	return append(env, "PATH="+newPath)
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
