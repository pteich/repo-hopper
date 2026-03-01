package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Settings holds all user-configurable options, persisted as JSON.
type Settings struct {
	ScanDirs          []string `json:"scan_dirs"`
	Editor            string   `json:"editor"`
	Terminal          string   `json:"terminal"`
	RescanIntervalMin int      `json:"rescan_interval_min"`
	MaxDepth          int      `json:"max_depth"`
	ExcludePatterns   []string `json:"exclude_patterns"`
	SortBy            string   `json:"sort_by"` // "last_commit" or "name"
	CheckForUpdates   bool     `json:"check_for_updates"`   // Whether to check for updates
	UpdateCheckHours int      `json:"update_check_hours"`  // How often to check (in hours, 0 = always)
	SkipUpdateVersion string   `json:"skip_update_version"` // Version to skip update prompt for
}

// ConfigDir returns the path to ~/.config/repo-hopper/
func ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "repo-hopper")
}

// ConfigFile returns the full path to the settings JSON file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "settings.json")
}

// DefaultSettings returns sensible defaults for all platforms.
func DefaultSettings() Settings {
	home, _ := os.UserHomeDir()

	terminal := "x-terminal-emulator"
	switch runtime.GOOS {
	case "darwin":
		terminal = "Terminal"
	case "windows":
		terminal = "cmd"
	}

	return Settings{
		ScanDirs:          []string{home},
		Editor:            "code",
		Terminal:          terminal,
		RescanIntervalMin: 5,
		MaxDepth:          6,
		ExcludePatterns:   []string{"node_modules", "vendor", ".cache", "Library"},
		SortBy:            "last_commit",
		CheckForUpdates:   true,
		UpdateCheckHours:  24, // Check daily by default
	}
}

// Load reads settings from disk. If the file doesn't exist, returns defaults.
func Load() Settings {
	data, err := os.ReadFile(ConfigFile())
	if err != nil {
		return DefaultSettings()
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return DefaultSettings()
	}

	// Fill in zero-value fields with defaults so partially-written configs still work
	defaults := DefaultSettings()
	if len(s.ScanDirs) == 0 {
		s.ScanDirs = defaults.ScanDirs
	}
	if s.Editor == "" {
		s.Editor = defaults.Editor
	}
	if s.Terminal == "" {
		s.Terminal = defaults.Terminal
	}
	if s.RescanIntervalMin <= 0 {
		s.RescanIntervalMin = defaults.RescanIntervalMin
	}
	if s.MaxDepth <= 0 {
		s.MaxDepth = defaults.MaxDepth
	}
	if s.SortBy == "" {
		s.SortBy = defaults.SortBy
	}

	return s
}

// Save writes the settings to disk as JSON.
func Save(s Settings) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigFile(), data, 0644)
}
