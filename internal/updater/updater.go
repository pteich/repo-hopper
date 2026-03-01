package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GitHubRelease represents a GitHub release API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	LatestVersion  string
	CurrentVersion string
	ReleaseURL     string
	ReleaseNotes   string
}

const (
	// repoOwner is the GitHub repository owner
	repoOwner = "pteich"
	// repoName is the GitHub repository name
	repoName = "repo-hopper"
	// apiURL is the GitHub API endpoint for releases
	apiURL = "https://api.github.com/repos/%s/%s/releases/latest"
)

// CheckForUpdates queries GitHub for the latest release and compares with current version
func CheckForUpdates(currentVersion string, skipVersion string) (*UpdateInfo, error) {
	if currentVersion == "" || strings.Contains(currentVersion, "unknown") {
		return nil, fmt.Errorf("current version not available")
	}

	// Fetch latest release from GitHub API
	url := fmt.Sprintf(apiURL, repoOwner, repoName)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	// Normalize versions for comparison (remove 'v' prefix if present)
	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(release.TagName, "v")

	// Check if user wants to skip this version
	if skipVersion != "" && strings.TrimPrefix(skipVersion, "v") == latest {
		return nil, nil // User chose to skip this version
	}

	// Check if there's actually an update available
	if current == latest {
		return nil, nil // Already up to date
	}

	// Ensure we return the full tag name for the latest version
	// and the original current version string
	return &UpdateInfo{
		LatestVersion:  release.TagName,
		CurrentVersion: currentVersion,
		ReleaseURL:     release.HTMLURL,
		ReleaseNotes:   release.Body,
	}, nil
}

// EnsureVersionPrefix returns the version string with a 'v' prefix if it doesn't already have one
// and is not "dev" or "unknown"
func EnsureVersionPrefix(v string) string {
	if v == "dev" || v == "unknown" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// GetCurrentVersion returns the version from ldflags, or "dev" if not set
func GetCurrentVersion() string {
	// This should be set via ldflags during build: -X main.version={{.Version}}
	// The variable will be in the main package
	return "unknown" // Will be set from main.version variable
}

// GetLastCheckTime returns when the last update check occurred
func GetLastCheckTime(configDir string) (time.Time, error) {
	filePath := configDir + "/last_update_check"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, string(data))
}

// SetLastCheckTime records when an update check occurred
func SetLastCheckTime(configDir string) error {
	filePath := configDir + "/last_update_check"
	data := time.Now().Format(time.RFC3339)
	return os.WriteFile(filePath, []byte(data), 0644)
}

// ShouldCheckForUpdates determines if an update check should run based on last check time
func ShouldCheckForUpdates(configDir string, checkIntervalHours int) (bool, error) {
	if checkIntervalHours <= 0 {
		return true, nil // Always check if interval is 0 or negative
	}

	lastCheck, err := GetLastCheckTime(configDir)
	if err != nil {
		return true, nil // Check if we can't read the file
	}

	return time.Since(lastCheck) > time.Duration(checkIntervalHours)*time.Hour, nil
}
