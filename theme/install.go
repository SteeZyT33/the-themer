package theme

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// InstallOpts configures the install operation.
type InstallOpts struct {
	HomeDir string // injectable for testing; defaults to os.UserHomeDir()
}

// resolveHome returns opts.HomeDir if set, otherwise os.UserHomeDir().
func (o InstallOpts) resolveHome() (string, error) {
	if o.HomeDir != "" {
		return o.HomeDir, nil
	}
	return os.UserHomeDir()
}

// InstallResult holds the outcome of installing one app's config.
type InstallResult struct {
	App     string
	Skipped bool   // true if the app dir didn't exist in the theme
	Message string // human-readable outcome
	Err     error
}

// Install deploys a theme's per-app configs to their filesystem destinations.
// It iterates all known app handlers, skipping any that the theme doesn't
// include. Errors are collected best-effort — every app is attempted.
func Install(t Theme, opts InstallOpts) []InstallResult {
	home, err := opts.resolveHome()
	if err != nil {
		return []InstallResult{{App: "home", Err: fmt.Errorf("resolving home directory: %w", err)}}
	}

	handlers := []struct {
		app     string
		install func(t Theme, home string) (string, error)
	}{
		{"ghostty", installGhostty},
		{"bat", installBat},
		{"delta", installDelta},
		{"fzf", installFzf},
		{"starship", installStarship},
		{"eza", installEza},
		{"gh-dash", installGhDash},
		{"vscode", installVscode},
	}

	var results []InstallResult
	for _, h := range handlers {
		appDir := filepath.Join(t.Dir, h.app)
		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			results = append(results, InstallResult{App: h.app, Skipped: true, Message: "no config in theme"})
			continue
		}

		msg, err := h.install(t, home)
		results = append(results, InstallResult{App: h.app, Message: msg, Err: err})
	}
	return results
}

// installGhostty copies theme files to ~/.config/ghostty/themes/.
func installGhostty(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "ghostty")
	destDir := filepath.Join(home, ".config", "ghostty", "themes")
	return copyDirContents(srcDir, destDir)
}

// installBat copies .tmTheme to bat's themes dir and rebuilds the cache.
func installBat(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "bat")

	// bat stores themes in $(bat --config-dir)/themes/
	destDir, err := batThemesDir(home)
	if err != nil {
		return "", err
	}

	msg, err := copyDirContents(srcDir, destDir)
	if err != nil {
		return msg, err
	}

	// Rebuild bat cache so the new theme is available.
	if batPath, lookErr := exec.LookPath("bat"); lookErr == nil {
		cmd := exec.Command(batPath, "cache", "--build")
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			return msg, fmt.Errorf("bat cache --build failed: %s: %w", strings.TrimSpace(string(out)), runErr)
		}
		msg += "; bat cache rebuilt"
	} else {
		msg += "; bat not on PATH, skipped cache rebuild"
	}
	return msg, nil
}

// batThemesDir returns the bat themes directory. Tries `bat --config-dir`
// first, falls back to ~/.config/bat/themes/.
func batThemesDir(home string) (string, error) {
	if batPath, err := exec.LookPath("bat"); err == nil {
		cmd := exec.Command(batPath, "--config-dir")
		out, err := cmd.Output()
		if err == nil {
			return filepath.Join(strings.TrimSpace(string(out)), "themes"), nil
		}
	}
	return filepath.Join(home, ".config", "bat", "themes"), nil
}

// installDelta copies gitconfig to ~/.config/the-themer/delta/ and adds
// an include.path entry to global git config if not already present.
func installDelta(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "delta")
	destDir := filepath.Join(home, ".config", "the-themer", "delta")

	msg, err := copyDirContents(srcDir, destDir)
	if err != nil {
		return msg, err
	}

	// Add include.path to global git config for each installed file.
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return msg, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		destPath := filepath.Join(destDir, e.Name())
		if err := gitAddIncludePath(destPath); err != nil {
			return msg, fmt.Errorf("adding git include.path for %s: %w", e.Name(), err)
		}
	}
	msg += "; git include.path configured"
	return msg, nil
}

// gitAddIncludePath adds path to git's global include.path if not already present.
func gitAddIncludePath(path string) error {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found: %w", err)
	}

	// Check existing include paths.
	cmd := exec.Command(gitPath, "config", "--global", "--get-all", "include.path")
	out, _ := cmd.Output() // exit 1 = no entries, that's fine
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == path {
			return nil // already present
		}
	}

	addCmd := exec.Command(gitPath, "config", "--global", "--add", "include.path", path)
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// installFzf copies fzf config to ~/.config/the-themer/fzf/.
func installFzf(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "fzf")
	destDir := filepath.Join(home, ".config", "the-themer", "fzf")
	return copyDirContents(srcDir, destDir)
}

// installStarship copies starship config to ~/.config/the-themer/starship/.
func installStarship(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "starship")
	destDir := filepath.Join(home, ".config", "the-themer", "starship")
	return copyDirContents(srcDir, destDir)
}

// installEza copies eza theme to ~/.config/eza/themes/.
func installEza(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "eza")
	destDir := filepath.Join(home, ".config", "eza", "themes")
	return copyDirContents(srcDir, destDir)
}

// installGhDash copies gh-dash config to ~/.config/the-themer/gh-dash/.
func installGhDash(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "gh-dash")
	destDir := filepath.Join(home, ".config", "the-themer", "gh-dash")
	return copyDirContents(srcDir, destDir)
}

// installVscode copies the VS Code / Cursor terminal theme to ~/.config/the-themer/vscode/.
func installVscode(t Theme, home string) (string, error) {
	srcDir := filepath.Join(t.Dir, "vscode")
	destDir := filepath.Join(home, ".config", "the-themer", "vscode")
	return copyDirContents(srcDir, destDir)
}

// copyDirContents copies all files from src to dest, creating dest if needed.
// Returns a summary message listing copied files.
func copyDirContents(srcDir, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", destDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", srcDir, err)
	}

	var copied, overwrote, unchanged []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dest := filepath.Join(destDir, e.Name())

		if _, err := os.Stat(dest); err == nil {
			if filesEqual(src, dest) {
				unchanged = append(unchanged, e.Name())
				continue
			}
			// File exists but differs — overwrite it.
			if err := copyFile(src, dest); err != nil {
				return "", fmt.Errorf("copying %s: %w", e.Name(), err)
			}
			overwrote = append(overwrote, e.Name())
			continue
		}

		if err := copyFile(src, dest); err != nil {
			return "", fmt.Errorf("copying %s: %w", e.Name(), err)
		}
		copied = append(copied, e.Name())
	}

	if len(copied) == 0 && len(overwrote) == 0 && len(unchanged) > 0 {
		return fmt.Sprintf("unchanged in %s", destDir), nil
	}

	// Build message from whichever lists have entries.
	var parts []string
	if len(copied) > 0 {
		parts = append(parts, fmt.Sprintf("installed %s", strings.Join(copied, ", ")))
	}
	if len(overwrote) > 0 {
		parts = append(parts, fmt.Sprintf("overwrote %s", strings.Join(overwrote, ", ")))
	}
	msg := fmt.Sprintf("%s to %s", strings.Join(parts, ", "), destDir)
	if len(unchanged) > 0 {
		msg += fmt.Sprintf(" (unchanged: %s)", strings.Join(unchanged, ", "))
	}
	return msg, nil
}

// copyFile copies a single file from src to dest, preserving permissions.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// filesEqual returns true if two files have identical SHA-256 hashes.
func filesEqual(a, b string) bool {
	hashA, errA := fileHash(a)
	hashB, errB := fileHash(b)
	if errA != nil || errB != nil {
		return false
	}
	return hashA == hashB
}

// fileHash returns the hex-encoded SHA-256 hash of a file's contents.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
