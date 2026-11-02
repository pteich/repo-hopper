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

	// OnChange, if set, is called after the scanner changes the state
	// (AddRepos, ClearRepos, SetScanning), possibly from another goroutine,
	// so a UI can redraw. Set it before scanning starts.
	OnChange func()
}

func (e *Engine) changed() {
	if e.OnChange != nil {
		e.OnChange()
	}
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
	e.filterLocked()

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
	defer e.changed() // runs after the unlock below
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state.AllRepos = append(e.state.AllRepos, repos...)

	// Re-apply search query to refresh the filtered view. Done inline
	// (not via UpdateSearch) because the lock is already held.
	e.filterLocked()

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
	defer e.changed() // runs after the unlock below
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.AllRepos = []Repo{}
	e.state.FilteredRepos = []Repo{}
	e.state.SelectedIndex = 0
}

// SetScanning updates the current scanning status
func (e *Engine) SetScanning(isScanning bool) {
	defer e.changed() // runs after the unlock below
	e.mu.Lock()
	defer e.mu.Unlock()
	e.state.IsScanning = isScanning
}

// SortRepos sorts a slice of repos by last modification (uncommitted edits
// win over the last commit) descending.
func SortRepos(repos []Repo) {
	sortRepos(repos, nil)
}

// sortRepos sorts by modification time descending, breaking ties by fuzzy
// match score (best first) when scores are given. Scores line up with repos.
func sortRepos(repos []Repo, scores []int) {
	sort.SliceStable(repos, func(i, j int) bool {
		mi, mj := repos[i].ModifiedTime(), repos[j].ModifiedTime()
		if !mi.Equal(mj) {
			return mi.After(mj)
		}
		return scores != nil && scores[i] > scores[j]
	})
}

// filterLocked recomputes FilteredRepos from AllRepos and the current query.
// The caller must hold e.mu. With a query, results are sorted by modification
// time first and fuzzy score only breaks ties.
func (e *Engine) filterLocked() {
	query := e.state.SearchQuery
	if query == "" {
		e.state.FilteredRepos = make([]Repo, len(e.state.AllRepos))
		copy(e.state.FilteredRepos, e.state.AllRepos)
		sortRepos(e.state.FilteredRepos, nil)
		return
	}

	repoNames := make([]string, len(e.state.AllRepos))
	for i, r := range e.state.AllRepos {
		repoNames[i] = r.Name + " " + r.Path
	}

	matches := fuzzy.Find(query, repoNames)

	filtered := make([]Repo, 0, len(matches))
	scores := make([]int, len(matches))
	for j, m := range matches {
		filtered = append(filtered, e.state.AllRepos[m.Index])
		scores[j] = m.Score
	}
	sortRepos(filtered, scores)
	e.state.FilteredRepos = filtered
}
