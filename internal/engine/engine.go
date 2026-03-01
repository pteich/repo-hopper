package engine

import (
	"sort"
	"sync"

	"github.com/sahilm/fuzzy"
)

// Engine is the central state holder and manager for repo-hopper.
type Engine struct {
	state State
	mu    sync.RWMutex
}

// New creates a new Engine with an initial empty state.
func New() *Engine {
	return &Engine{
		state: State{
			AllRepos:      []Repo{},
			FilteredRepos: []Repo{},
			SearchQuery:   "",
			SelectedIndex: 0,
			IsScanning:    false,
		},
	}
}

// GetState returns a copy of the current state safely.
func (e *Engine) GetState() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// UpdateSearch updates the filter query.
// It applies fuzzy searching to AllRepos and updates FilteredRepos and SelectedIndex.
func (e *Engine) UpdateSearch(query string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state.SearchQuery = query

	if query == "" {
		e.state.FilteredRepos = make([]Repo, len(e.state.AllRepos))
		copy(e.state.FilteredRepos, e.state.AllRepos)
		SortRepos(e.state.FilteredRepos)
	} else {
		repoNames := make([]string, len(e.state.AllRepos))
		for i, r := range e.state.AllRepos {
			repoNames[i] = r.Name + " " + r.Path
		}

		matches := fuzzy.Find(query, repoNames)

		e.state.FilteredRepos = make([]Repo, 0, len(matches))
		for _, m := range matches {
			e.state.FilteredRepos = append(e.state.FilteredRepos, e.state.AllRepos[m.Index])
		}
		// Sort by recently changed rather than fuzzy match score
		// TODO make this configurable (fuzzy match score vs recently changed)
		SortRepos(e.state.FilteredRepos)
	}

	// Reset index on new search
	if len(e.state.FilteredRepos) > 0 {
		e.state.SelectedIndex = 0
	} else {
		e.state.SelectedIndex = -1
	}
}

// SelectNext moves the selection down in the list.
func (e *Engine) SelectNext() {
	e.mu.Lock()
	defer e.mu.Unlock()

	total := len(e.state.FilteredRepos)
	if total == 0 {
		return
	}

	e.state.SelectedIndex++
	if e.state.SelectedIndex >= total {
		e.state.SelectedIndex = total - 1 // clamp to bottom
	}
}

// SelectPrev moves the selection up in the list.
func (e *Engine) SelectPrev() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.state.FilteredRepos) == 0 {
		return
	}

	e.state.SelectedIndex--
	if e.state.SelectedIndex < 0 {
		e.state.SelectedIndex = 0 // clamp to top
	}
}

// SetSelectedIndex directly sets the selection index based on user clicks.
func (e *Engine) SetSelectedIndex(index int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	total := len(e.state.FilteredRepos)
	if total == 0 {
		return
	}

	if index >= 0 && index < total {
		e.state.SelectedIndex = index
	}
}

// AddRepos allows adding newly discovered repos to the engine safely.
func (e *Engine) AddRepos(repos []Repo) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state.AllRepos = append(e.state.AllRepos, repos...)

	// Re-apply search query to refresh the filtered view
	query := e.state.SearchQuery

	// Temporarily unlock to call UpdateSearch (which acquires lock), or just do it inline
	// Inline is safer here
	if query == "" {
		e.state.FilteredRepos = make([]Repo, len(e.state.AllRepos))
		copy(e.state.FilteredRepos, e.state.AllRepos)
		SortRepos(e.state.FilteredRepos)
	} else {
		repoNames := make([]string, len(e.state.AllRepos))
		for i, r := range e.state.AllRepos {
			repoNames[i] = r.Name + " " + r.Path
		}
		matches := fuzzy.Find(query, repoNames)
		e.state.FilteredRepos = make([]Repo, 0, len(matches))
		for _, m := range matches {
			e.state.FilteredRepos = append(e.state.FilteredRepos, e.state.AllRepos[m.Index])
		}
		SortRepos(e.state.FilteredRepos)
	}

	// Safety check for SelectedIndex
	if len(e.state.FilteredRepos) == 0 {
		e.state.SelectedIndex = -1
	} else if e.state.SelectedIndex >= len(e.state.FilteredRepos) {
		e.state.SelectedIndex = len(e.state.FilteredRepos) - 1
	} else if e.state.SelectedIndex == -1 && len(e.state.FilteredRepos) > 0 {
		e.state.SelectedIndex = 0
	}
}

// ClearRepos resets the repo list, used before a re-scan.
func (e *Engine) ClearRepos() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.AllRepos = []Repo{}
	e.state.FilteredRepos = []Repo{}
	e.state.SelectedIndex = 0
}

// SetScanning updates the current scanning status
func (e *Engine) SetScanning(isScanning bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.IsScanning = isScanning
}

// SortRepos sorts a slice of repos by LastCommitTime descending.
func SortRepos(repos []Repo) {
	sort.SliceStable(repos, func(i, j int) bool {
		return repos[i].LastCommitTime.After(repos[j].LastCommitTime)
	})
}
