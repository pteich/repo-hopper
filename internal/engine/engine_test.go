package engine

import (
	"testing"
	"time"
)

// TestEngineSortByModifiedTime checks that dirty repos sort above repos whose
// last commit is older, and that fuzzy score only breaks time ties.
func TestEngineSortByModifiedTime(t *testing.T) {
	now := time.Now()
	eng := New()

	repos := []Repo{
		{Name: "clean-old", Path: "/a/clean-old", LastCommitTime: now.Add(-48 * time.Hour)},
		{Name: "dirty-recent", Path: "/a/dirty-recent", LastCommitTime: now.Add(-48 * time.Hour),
			Dirty: true, DirtyTime: now.Add(-time.Hour)},
		{Name: "clean-recent", Path: "/a/clean-recent", LastCommitTime: now},
	}
	eng.AddRepos(repos)

	got := eng.GetState().FilteredRepos
	want := []string{"clean-recent", "dirty-recent", "clean-old"}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("position %d: expected %s, got %s", i, name, got[i].Name)
		}
	}
}

// TestEngineComputeAgeDirty checks the cached age reflects uncommitted edits.
func TestEngineComputeAgeDirty(t *testing.T) {
	now := time.Now()
	r := Repo{
		LastCommitTime: now.Add(-48 * time.Hour),
		Dirty:          true,
		DirtyTime:      now.Add(-30 * time.Minute),
	}
	r.ComputeAge()
	if r.CachedAge != "30 mins" {
		t.Errorf("expected age from dirty time, got %q", r.CachedAge)
	}
}

func TestEngineInitialState(t *testing.T) {
	eng := New()
	state := eng.GetState()

	if len(state.AllRepos) != 0 {
		t.Errorf("Expected 0 repos, got %d", len(state.AllRepos))
	}
	if state.SearchQuery != "" {
		t.Errorf("Expected empty search query")
	}
	if state.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex 0, got %d", state.SelectedIndex)
	}
}

func TestEngineAddRepos(t *testing.T) {
	eng := New()
	now := time.Now()

	repos := []Repo{
		{Name: "repo1", Path: "/a/repo1", LastCommitTime: now},
		{Name: "repo2", Path: "/a/repo2", LastCommitTime: now.Add(-time.Hour)},
	}

	eng.AddRepos(repos)
	state := eng.GetState()

	if len(state.AllRepos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(state.AllRepos))
	}
	// AddRepos auto-sorts FilteredRepos if query is empty
	if state.FilteredRepos[0].Name != "repo1" {
		t.Errorf("Expected repo1 to be first (most recent)")
	}
}

func TestEngineFuzzySearch(t *testing.T) {
	eng := New()
	now := time.Now()

	repos := []Repo{
		{Name: "web-frontend", Path: "/dev/web-frontend"},
		{Name: "api-backend", Path: "/dev/api-backend"},
		{Name: "shared-core", Path: "/dev/shared-core"},
	}

	for i := range repos {
		repos[i].LastCommitTime = now.Add(time.Duration(-i) * time.Hour) // just pseudo sorting
	}

	eng.AddRepos(repos)

	// Test exact match
	eng.UpdateSearch("web")
	state := eng.GetState()
	if len(state.FilteredRepos) != 1 || state.FilteredRepos[0].Name != "web-frontend" {
		t.Errorf("Search 'web' failed")
	}

	// Test fuzzy match ('b' and 'k' -> api-backend)
	eng.UpdateSearch("bek")
	state = eng.GetState()
	if len(state.FilteredRepos) != 1 || state.FilteredRepos[0].Name != "api-backend" {
		t.Errorf("Fuzzy search 'bek' failed")
	}

	// Test finding path
	eng.UpdateSearch("dev")
	state = eng.GetState()
	if len(state.FilteredRepos) != 3 {
		t.Errorf("Expected 3 results for query 'dev' matching path, got %d", len(state.FilteredRepos))
	}
}

func TestEngineSelection(t *testing.T) {
	eng := New()
	repos := []Repo{
		{Name: "A", LastCommitTime: time.Now()},
		{Name: "B", LastCommitTime: time.Now().Add(-1 * time.Minute)},
		{Name: "C", LastCommitTime: time.Now().Add(-2 * time.Minute)},
	}
	eng.AddRepos(repos)

	eng.SelectNext() // should be index 1
	if eng.GetState().SelectedIndex != 1 {
		t.Errorf("Expected index 1")
	}

	eng.SelectNext() // should be index 2
	eng.SelectNext() // should still be index 2 (clamp)
	if eng.GetState().SelectedIndex != 2 {
		t.Errorf("Expected index 2 (clamped)")
	}

	eng.SelectPrev() // should be index 1
	if eng.GetState().SelectedIndex != 1 {
		t.Errorf("Expected index 1")
	}

	// Test search resets index
	eng.UpdateSearch("C")
	if eng.GetState().SelectedIndex != 0 {
		t.Errorf("Expected search to reset index to 0")
	}
}
