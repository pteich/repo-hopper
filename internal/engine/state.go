package engine

import (
	"fmt"
	"time"
)

// Repo represents a discovered Git repository.
type Repo struct {
	Path           string    `json:"path"`
	Name           string    `json:"name"`
	LastCommitMsg  string    `json:"lastCommitMsg"`
	LastCommitTime time.Time `json:"lastCommitTime"`
	Branch         string    `json:"branch"`
	CachedAge      string    `json:"-"`
}

// ComputeAge computes and sets the CachedAge field for the repo.
func (r *Repo) ComputeAge() string {
	if r.LastCommitTime.IsZero() {
		r.CachedAge = "No commits"
		return r.CachedAge
	}

	now := time.Now()
	d := now.Sub(r.LastCommitTime)

	switch {
	case d < time.Minute:
		r.CachedAge = "Just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			r.CachedAge = "1 min"
		} else {
			r.CachedAge = fmt.Sprintf("%d mins", mins)
		}
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			r.CachedAge = "1 hr"
		} else {
			r.CachedAge = fmt.Sprintf("%d hrs", hours)
		}
	case d < 48*time.Hour:
		r.CachedAge = "Yesterday"
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		r.CachedAge = fmt.Sprintf("%d days", days)
	default:
		r.CachedAge = r.LastCommitTime.Format("Jan 2, 2006")
	}
	return r.CachedAge
}

// State represents the entire UI-agnostic application state.
// A UI framework like Bubbletea or Shirei can directly consume this.
type State struct {
	// AllRepos contains all repositories found by the scanner.
	AllRepos []Repo `json:"allRepos"`

	// FilteredRepos contains only the repos that match the current SearchQuery.
	// If SearchQuery is empty, this is identical to AllRepos.
	FilteredRepos []Repo `json:"filteredRepos"`

	// SearchQuery is the current user input to filter the list.
	SearchQuery string `json:"searchQuery"`

	// SelectedIndex is the index of the currently highlighted repo in FilteredRepos.
	SelectedIndex int `json:"selectedIndex"`

	// IsScanning indicates whether a background directory scan is currently running.
	IsScanning bool `json:"isScanning"`
}
