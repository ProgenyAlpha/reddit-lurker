package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
)

const (
	repoOwner = "ProgenyAlpha"
	repoName  = "reddit-lurker"
)

// Update handles the "lurk update" subcommand.
func Update(currentVersion string, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	checkOnly := fs.Bool("check", false, "Check for updates without installing")
	force := fs.Bool("force", false, "Force update even with dev version")
	fs.Parse(args)

	if currentVersion == "dev" && !*force {
		fmt.Fprintln(os.Stderr, "Cannot determine current version (dev build). Use --force to update anyway.")
		os.Exit(1)
	}

	method := detectInstallMethod()
	if method != "" {
		fmt.Fprintf(os.Stderr, "lurk was installed via %s\n", method)
		fmt.Fprintf(os.Stderr, "Update with: %s\n", updateCommand(method))
		os.Exit(0)
	}

	// Check write permission to binary directory
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error locating binary: %s\n", err)
		os.Exit(1)
	}
	if err := checkWritePermission(filepath.Dir(exe)); err != nil {
		fmt.Fprintf(os.Stderr, "No write permission to %s: %s\n", filepath.Dir(exe), err)
		fmt.Fprintln(os.Stderr, "Try running with sudo or moving the binary to a writable location.")
		os.Exit(1)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing updater: %s\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %s\n", err)
		os.Exit(1)
	}
	if !found {
		fmt.Fprintln(os.Stderr, "No releases found.")
		os.Exit(1)
	}

	v := currentVersion
	if *force && v == "dev" {
		v = "0.0.0"
	}

	if latest.LessOrEqual(v) {
		fmt.Fprintf(os.Stderr, "lurk v%s is already the latest version.\n", currentVersion)
		return
	}

	if *checkOnly {
		fmt.Fprintf(os.Stderr, "lurk v%s is available (current: v%s). Run 'lurk update' to upgrade.\n", latest.Version(), currentVersion)
		return
	}

	dlCtx, dlCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer dlCancel()

	if err := updater.UpdateTo(dlCtx, latest, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Updated lurk from v%s to v%s\n", currentVersion, latest.Version())
}

// CheckForUpdate runs a non-blocking background version check.
// It caches results to avoid hitting GitHub API rate limits.
func CheckForUpdate(currentVersion string) {
	if currentVersion == "dev" {
		return
	}

	configDir := configPath()
	lastCheckFile := filepath.Join(configDir, "last-check")

	// Check if we've checked recently (within 24 hours)
	if data, err := os.ReadFile(lastCheckFile); err == nil {
		if ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			if time.Since(time.Unix(ts, 0)) < 24*time.Hour {
				return
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil || !found {
		return
	}

	// Write last-check timestamp
	os.MkdirAll(configDir, 0755)
	os.WriteFile(lastCheckFile, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)

	// If newer version available, write it to update-available file
	if latest.GreaterThan(currentVersion) {
		os.WriteFile(filepath.Join(configDir, "update-available"), []byte(latest.Version()), 0644)
	} else {
		// Remove stale notice if we're already up to date
		os.Remove(filepath.Join(configDir, "update-available"))
	}
}

// PrintUpdateNotice prints a one-line update notice to stderr if available.
func PrintUpdateNotice(currentVersion string) {
	if currentVersion == "dev" {
		return
	}

	availableFile := filepath.Join(configPath(), "update-available")
	data, err := os.ReadFile(availableFile)
	if err != nil {
		return
	}

	newVersion := strings.TrimSpace(string(data))
	if newVersion == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "\nlurk v%s is available. Run 'lurk update' to upgrade.\n", newVersion)
	os.Remove(availableFile)
}

// detectInstallMethod checks how lurk was installed.
func detectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}

	switch {
	case strings.Contains(resolved, "/node_modules/") || strings.Contains(resolved, "/lib/node_modules/"):
		return "npm"
	case strings.Contains(resolved, "/Cellar/") || strings.Contains(resolved, "/homebrew/"):
		return "brew"
	case isGoInstall(resolved):
		return "go"
	}

	return ""
}

// isGoInstall checks if the binary is in GOPATH/bin or GOBIN.
func isGoInstall(path string) bool {
	binDir := filepath.Dir(filepath.Clean(path))

	if gobin := os.Getenv("GOBIN"); gobin != "" {
		if binDir == filepath.Clean(gobin) {
			return true
		}
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		if binDir == filepath.Join(gopath, "bin") {
			return true
		}
	}
	// Default GOPATH is ~/go
	home, err := os.UserHomeDir()
	if err == nil {
		if binDir == filepath.Join(home, "go", "bin") {
			return true
		}
	}
	return false
}

// updateCommand returns the appropriate update command for a package manager.
func updateCommand(method string) string {
	switch method {
	case "npm":
		return "npm update -g reddit-lurker"
	case "brew":
		return "brew upgrade reddit-lurker"
	case "go":
		return "go install github.com/ProgenyAlpha/reddit-lurker@latest"
	}
	return ""
}

// checkWritePermission verifies we can write to a directory.
func checkWritePermission(dir string) error {
	f, err := os.CreateTemp(dir, ".lurk-update-check-*")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Remove(name)
}

// configPath returns the lurk config directory path.
func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lurk")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "lurk")
	}
	return filepath.Join(home, ".config", "lurk")
}
