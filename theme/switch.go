package theme

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SwitchOpts configures the switch operation.
type SwitchOpts struct {
	HomeDir string // injectable for testing; defaults to os.UserHomeDir()
}

// resolveHome returns opts.HomeDir if set, otherwise os.UserHomeDir().
func (o SwitchOpts) resolveHome() (string, error) {
	if o.HomeDir != "" {
		return o.HomeDir, nil
	}
	return os.UserHomeDir()
}

// SwitchResult holds the outcome of switching one app.
type SwitchResult struct {
	App     string
	Skipped bool
	Message string
	Err     error
}

// Switch activates a theme across all supported apps. Each app is handled
// independently — errors are collected best-effort.
func Switch(t Theme, opts SwitchOpts) []SwitchResult {
	home, err := opts.resolveHome()
	if err != nil {
		return []SwitchResult{{App: "home", Err: fmt.Errorf("resolving home directory: %w", err)}}
	}

	handlers := []struct {
		app     string
		switch_ func(t Theme, home string) (string, error)
	}{
		{"ghostty", switchGhostty},
		{"bat", switchBat},
		{"delta", switchDelta},
		{"fzf", switchFzf},
		{"starship", switchStarship},
		{"eza", switchEza},
		{"gh-dash", switchGhDash},
		{"neovim", switchNeovim},
		{"claude", switchClaude},
		{"vscode", switchVscode},
		{"windows-terminal", switchWindowsTerminal},
	}

	var results []SwitchResult
	for _, h := range handlers {
		msg, err := h.switch_(t, home)
		if msg == "" && err == nil {
			// Handler signaled nothing to do.
			results = append(results, SwitchResult{App: h.app, Skipped: true, Message: "not configured for this theme"})
			continue
		}
		results = append(results, SwitchResult{App: h.app, Message: msg, Err: err})
	}
	return results
}

// switchGhostty writes theme.local with the theme filename reference.
// Ghostty matches the `theme` value against filenames in its themes directory,
// so we use the full filename including any .ghostty extension.
func switchGhostty(t Theme, home string) (string, error) {
	ghosttyDir := filepath.Join(t.Dir, "ghostty")
	if _, err := os.Stat(ghosttyDir); os.IsNotExist(err) {
		return "", nil
	}

	themeFile, err := firstFile(ghosttyDir)
	if err != nil {
		return "", fmt.Errorf("reading ghostty dir: %w", err)
	}
	if themeFile == "" {
		return "", nil
	}

	content := fmt.Sprintf("# Managed by the-themer — do not edit\ntheme = %s\n", themeFile)
	dest := filepath.Join(home, ".config", "ghostty", "theme.local")

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("theme.local -> %s", themeFile), nil
}

// switchBat writes the bat theme name to bat-theme.txt.
// bat identifies custom themes by filename (sans .tmTheme extension), so we
// read the actual filename from the theme's bat/ directory.
// If only a reference is set, use that directly.
func switchBat(t Theme, home string) (string, error) {
	batDir := filepath.Join(t.Dir, "bat")
	hasBatDir := dirExists(batDir)
	refName := t.Config.References["bat"]

	if !hasBatDir && refName == "" {
		return "", nil
	}

	var themeName string
	if hasBatDir {
		file, err := firstFile(batDir)
		if err != nil {
			return "", fmt.Errorf("reading bat dir: %w", err)
		}
		if file == "" {
			return "", nil
		}
		themeName = strings.TrimSuffix(file, ".tmTheme")
	} else {
		themeName = refName
	}

	dest := filepath.Join(home, ".config", "bat-theme.txt")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, []byte(themeName+"\n"), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("bat-theme.txt -> %s", themeName), nil
}

// switchDelta writes the delta feature name to delta-theme.txt.
// The feature name is derived from the gitconfig filename (without extension).
func switchDelta(t Theme, home string) (string, error) {
	deltaDir := filepath.Join(t.Dir, "delta")
	hasDeltaDir := dirExists(deltaDir)
	refName := t.Config.References["delta"]

	if !hasDeltaDir && refName == "" {
		return "", nil
	}

	var featureName string
	if hasDeltaDir {
		// The gitconfig filename (without .gitconfig) is the delta feature name.
		file, err := firstFile(deltaDir)
		if err != nil {
			return "", err
		}
		featureName = strings.TrimSuffix(file, ".gitconfig")
	} else {
		featureName = refName
	}

	dest := filepath.Join(home, ".config", "delta-theme.txt")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, []byte(featureName+"\n"), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("delta-theme.txt -> %s", featureName), nil
}

// switchFzf creates a symlink current.zsh pointing to the installed fzf config.
func switchFzf(t Theme, home string) (string, error) {
	fzfDir := filepath.Join(t.Dir, "fzf")
	if !dirExists(fzfDir) {
		return "", nil
	}

	srcFile, err := firstFile(fzfDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	installedDir := filepath.Join(home, ".config", "the-themer", "fzf")
	installedFile := filepath.Join(installedDir, srcFile)
	link := filepath.Join(installedDir, "current.zsh")

	// Remove existing symlink before creating new one.
	os.Remove(link)
	if err := os.Symlink(installedFile, link); err != nil {
		return "", err
	}
	return fmt.Sprintf("fzf/current.zsh -> %s", srcFile), nil
}

// switchStarship symlinks ~/.config/starship.toml to the installed starship config.
func switchStarship(t Theme, home string) (string, error) {
	starshipDir := filepath.Join(t.Dir, "starship")
	if !dirExists(starshipDir) {
		return "", nil
	}

	srcFile, err := firstFile(starshipDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	installedFile := filepath.Join(home, ".config", "the-themer", "starship", srcFile)
	link := filepath.Join(home, ".config", "starship.toml")

	os.Remove(link)
	if err := os.Symlink(installedFile, link); err != nil {
		return "", err
	}
	return fmt.Sprintf("starship.toml -> %s", srcFile), nil
}

// switchEza symlinks ~/.config/eza/theme.yml to the installed eza theme.
func switchEza(t Theme, home string) (string, error) {
	ezaDir := filepath.Join(t.Dir, "eza")
	if !dirExists(ezaDir) {
		return "", nil
	}

	srcFile, err := firstFile(ezaDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	installedFile := filepath.Join(home, ".config", "eza", "themes", srcFile)
	link := filepath.Join(home, ".config", "eza", "theme.yml")

	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return "", err
	}
	os.Remove(link)
	if err := os.Symlink(installedFile, link); err != nil {
		return "", err
	}
	return fmt.Sprintf("eza/theme.yml -> %s", srcFile), nil
}

// switchGhDash copies the installed gh-dash config to ~/.config/gh-dash/config.yml.
func switchGhDash(t Theme, home string) (string, error) {
	ghDashDir := filepath.Join(t.Dir, "gh-dash")
	if !dirExists(ghDashDir) {
		return "", nil
	}

	srcFile, err := firstFile(ghDashDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	src := filepath.Join(home, ".config", "the-themer", "gh-dash", srcFile)
	dest := filepath.Join(home, ".config", "gh-dash", "config.yml")

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := copyFile(src, dest); err != nil {
		return "", err
	}
	return fmt.Sprintf("gh-dash/config.yml -> %s", srcFile), nil
}

// switchNeovim uses headless nvim to set the colorscheme via Themery.
func switchNeovim(t Theme, home string) (string, error) {
	name := t.Config.References["neovim"]
	if name == "" {
		return "", nil
	}

	nvimPath, err := exec.LookPath("nvim")
	if err != nil {
		return "nvim not on PATH, skipped", nil
	}

	luaCmd := fmt.Sprintf(`pcall(function() require('themery').setThemeByName('%s', true) end)`, name)
	cmd := exec.Command(nvimPath, "--headless", "-c", fmt.Sprintf("lua %s", luaCmd), "-c", "qa")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("nvim themery switch failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return fmt.Sprintf("neovim -> %s", name), nil
}

// switchClaude edits ~/.claude.json to set the theme key.
// "dark" deletes the key (dark is the default).
// Uses sjson for surgical edits — only the theme key is touched, preserving
// key order, formatting, and numeric precision in the rest of the file.
// Atomic write (temp file + rename) and original permissions are preserved.
func switchClaude(t Theme, home string) (string, error) {
	value := t.Config.References["claude"]
	if value == "" {
		return "", nil
	}

	claudePath := filepath.Join(home, ".claude.json")

	var origPerm os.FileMode = 0o644
	raw, err := os.ReadFile(claudePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("reading claude.json: %w", err)
		}
		// File doesn't exist — start with empty object.
		raw = []byte("{}")
	} else if info, err := os.Stat(claudePath); err == nil {
		origPerm = info.Mode().Perm()
	}

	var out []byte
	if value == "dark" {
		out, err = sjson.DeleteBytes(raw, "theme")
	} else {
		out, err = sjson.SetBytes(raw, "theme", value)
	}
	if err != nil {
		return "", fmt.Errorf("editing claude.json: %w", err)
	}

	// Atomic write: temp file in same directory, then rename.
	tmpFile, err := os.CreateTemp(filepath.Dir(claudePath), ".claude.json.tmp.*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(out); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, origPerm); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("setting permissions: %w", err)
	}
	if err := os.Rename(tmpPath, claudePath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	if value == "dark" {
		return "claude.json -> removed theme key (dark is default)", nil
	}
	return fmt.Sprintf("claude.json -> %s", value), nil
}

// switchVscode merges terminal color customizations into VS Code / Cursor settings.json.
// It reads the generated .jsonc file, strips comments, and sets each key under
// "workbench.colorCustomizations" in the user's settings.json.
// Supports native Linux, macOS, and WSL (auto-detects Windows paths via wslpath).
func switchVscode(t Theme, home string) (string, error) {
	vscodeDir := filepath.Join(t.Dir, "vscode")
	if !dirExists(vscodeDir) {
		return "", nil
	}

	srcFile, err := firstFile(vscodeDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	// Read the generated theme JSONC.
	installedFile := filepath.Join(home, ".config", "the-themer", "vscode", srcFile)
	themeData, err := os.ReadFile(installedFile)
	if err != nil {
		return "", fmt.Errorf("reading vscode theme: %w", err)
	}

	pairs := parseJSONCKeyValues(string(themeData))

	settingsPath, err := findVscodeSettings(home)
	if err != nil {
		return "", err
	}

	if err := mergeVscodeSettings(settingsPath, pairs); err != nil {
		return "", err
	}

	return fmt.Sprintf("settings.json -> %d terminal color keys (%s)", len(pairs), settingsPath), nil
}

// switchWindowsTerminal adds or updates the color scheme in Windows Terminal settings.json
// and sets it as the default colorScheme. Detects WSL and finds the Windows-side settings.
func switchWindowsTerminal(t Theme, home string) (string, error) {
	wtDir := filepath.Join(t.Dir, "windows-terminal")
	if !dirExists(wtDir) {
		return "", nil
	}

	srcFile, err := firstFile(wtDir)
	if err != nil || srcFile == "" {
		return "", err
	}

	// Read the generated scheme JSON.
	installedFile := filepath.Join(home, ".config", "the-themer", "windows-terminal", srcFile)
	schemeData, err := os.ReadFile(installedFile)
	if err != nil {
		return "", fmt.Errorf("reading windows-terminal theme: %w", err)
	}

	settingsPath, err := findWTSettings(home)
	if err != nil {
		return "", err
	}
	if settingsPath == "" {
		return "Windows Terminal settings.json not found, skipped", nil
	}

	var origPerm os.FileMode = 0o644
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return "", fmt.Errorf("reading WT settings.json: %w", err)
	}
	if info, err := os.Stat(settingsPath); err == nil {
		origPerm = info.Mode().Perm()
	}

	// Strip JSON comments (Windows Terminal settings can have // comments).
	cleanRaw := stripJSONComments(string(raw))

	// Use gjson to find existing schemes and sjson to update.
	themeName := t.Config.Theme.Name

	// Remove existing scheme with same name if present.
	schemes := gjson.Get(cleanRaw, "schemes")
	newSchemes := "["
	first := true
	if schemes.Exists() && schemes.IsArray() {
		schemes.ForEach(func(_, value gjson.Result) bool {
			if value.Get("name").String() != themeName {
				if !first {
					newSchemes += ","
				}
				newSchemes += value.Raw
				first = false
			}
			return true
		})
	}
	// Append the new scheme.
	if !first {
		newSchemes += ","
	}
	newSchemes += strings.TrimSpace(string(schemeData))
	newSchemes += "]"

	// Set schemes array.
	out, err := sjson.SetRawBytes([]byte(cleanRaw), "schemes", []byte(newSchemes))
	if err != nil {
		return "", fmt.Errorf("setting schemes: %w", err)
	}

	// Set default profile colorScheme.
	out, err = sjson.SetBytes(out, "profiles.defaults.colorScheme", themeName)
	if err != nil {
		return "", fmt.Errorf("setting default colorScheme: %w", err)
	}

	// Atomic write.
	tmpFile, err := os.CreateTemp(filepath.Dir(settingsPath), ".wt-settings.tmp.*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(out); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(tmpPath, origPerm); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return fmt.Sprintf("WT settings.json -> scheme %q set as default (%s)", themeName, settingsPath), nil
}

// findWTSettings locates Windows Terminal's settings.json.
// On WSL, probes the Windows-side LocalAppData paths.
func findWTSettings(home string) (string, error) {
	// Native Linux path (unlikely but check anyway).
	native := filepath.Join(home, ".config", "windows-terminal", "settings.json")
	if _, err := os.Stat(native); err == nil {
		return native, nil
	}

	if !isWSL() {
		return "", nil
	}

	localAppData, err := wslLocalAppData()
	if err != nil {
		return "", nil // Not an error — just no WT to configure.
	}

	candidates := []string{
		// Standard Windows Terminal (Microsoft Store).
		filepath.Join(localAppData, "Packages", "Microsoft.WindowsTerminal_8wekyb3d8bbwe", "LocalState", "settings.json"),
		// Windows Terminal Preview.
		filepath.Join(localAppData, "Packages", "Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe", "LocalState", "settings.json"),
		// Portable / non-store version.
		filepath.Join(localAppData, "Microsoft", "Windows Terminal", "settings.json"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", nil
}

// wslLocalAppData returns the WSL-mounted path to Windows %LOCALAPPDATA%.
func wslLocalAppData() (string, error) {
	cmd := exec.Command("cmd.exe", "/C", "echo", "%LOCALAPPDATA%")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	winPath := strings.TrimSpace(strings.ReplaceAll(string(out), "\r", ""))
	if winPath == "" || winPath == "%LOCALAPPDATA%" {
		return "", fmt.Errorf("LOCALAPPDATA not set")
	}

	wsl := exec.Command("wslpath", winPath)
	wslOut, err := wsl.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(wslOut)), nil
}

// stripJSONComments removes // line comments from JSON (Windows Terminal uses them).
func stripJSONComments(s string) string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// vscodeKV holds a parsed key-value pair from the generated JSONC.
type vscodeKV struct {
	key   string
	value string
}

// parseJSONCKeyValues strips comments from JSONC and extracts flat key-value pairs.
func parseJSONCKeyValues(data string) []vscodeKV {
	var pairs []vscodeKV
	for _, line := range strings.Split(data, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		trimmed = strings.TrimSuffix(trimmed, ",")
		if !strings.Contains(trimmed, ":") || trimmed == "{" || trimmed == "}" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(parts[0]), "\"")
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		pairs = append(pairs, vscodeKV{key, value})
	}
	return pairs
}

// findVscodeSettings locates the VS Code / Cursor settings.json.
// Checks native Linux/macOS paths first, then detects WSL and probes
// the Windows-side AppData paths via wslpath.
func findVscodeSettings(home string) (string, error) {
	// Native paths (Linux and macOS).
	candidates := []string{
		filepath.Join(home, ".config", "Cursor", "User", "settings.json"),
		filepath.Join(home, ".config", "Code", "User", "settings.json"),
		filepath.Join(home, "Library", "Application Support", "Cursor", "User", "settings.json"),
		filepath.Join(home, "Library", "Application Support", "Code", "User", "settings.json"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// WSL detection: check for Windows AppData paths.
	if isWSL() {
		appData, err := wslAppData()
		if err == nil && appData != "" {
			wslCandidates := []string{
				filepath.Join(appData, "Cursor", "User", "settings.json"),
				filepath.Join(appData, "Code", "User", "settings.json"),
			}
			for _, p := range wslCandidates {
				if _, err := os.Stat(p); err == nil {
					return p, nil
				}
			}
		}
	}

	// Default to Cursor native path (will be created if needed).
	return candidates[0], nil
}

// isWSL returns true if running under Windows Subsystem for Linux.
func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

// wslAppData returns the WSL-mounted path to Windows %APPDATA%.
// Uses cmd.exe to read the environment variable, then wslpath to convert.
func wslAppData() (string, error) {
	cmd := exec.Command("cmd.exe", "/C", "echo", "%APPDATA%")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	winPath := strings.TrimSpace(strings.ReplaceAll(string(out), "\r", ""))
	if winPath == "" || winPath == "%APPDATA%" {
		return "", fmt.Errorf("APPDATA not set")
	}

	wsl := exec.Command("wslpath", winPath)
	wslOut, err := wsl.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(wslOut)), nil
}

// mergeVscodeSettings reads settings.json, sets each key under
// workbench.colorCustomizations, and writes back atomically.
func mergeVscodeSettings(settingsPath string, pairs []vscodeKV) error {
	var origPerm os.FileMode = 0o644
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading settings.json: %w", err)
		}
		raw = []byte("{}")
	} else if info, err := os.Stat(settingsPath); err == nil {
		origPerm = info.Mode().Perm()
	}

	out := raw
	for _, p := range pairs {
		// Escape dots in the key so sjson treats "terminal.background" as a
		// literal key, not a nested path. VS Code expects flat dotted keys.
		escapedKey := strings.ReplaceAll(p.key, ".", "\\.")
		path := "workbench\\.colorCustomizations." + escapedKey
		out, err = sjson.SetBytes(out, path, p.value)
		if err != nil {
			return fmt.Errorf("setting %s: %w", p.key, err)
		}
	}

	// Atomic write.
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(settingsPath), ".settings.json.tmp.*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(out); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, origPerm); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// firstFile returns the name of the first regular file in dir, or "" if empty.
func firstFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			return e.Name(), nil
		}
	}
	return "", nil
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
