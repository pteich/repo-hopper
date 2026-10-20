package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
)

var now = time.Now()

// newTester resets the view state to the repos alpha, beta, gamma (newest
// first) and returns a tester with the search field focused.
func newTester(t *testing.T) *ui.Tester {
	t.Helper()
	eng = engine.New()
	settings = &config.Settings{}
	query, chosen, showSettings = "", 0, false
	list = ui.ListState{}
	search = ui.Handle{}
	eng.AddRepos([]engine.Repo{
		{Name: "alpha", Path: "/src/alpha", LastCommitTime: now},
		{Name: "beta", Path: "/src/beta", LastCommitTime: now.Add(-time.Hour)},
		{Name: "gamma", Path: "/src/gamma", LastCommitTime: now.Add(-2 * time.Hour)},
	})
	tt := ui.NewTester(view, 800, 600)
	search.Focus()
	tt.Frame()
	if !tt.HasText("alpha") || !tt.HasText("3 repositories") {
		t.Fatalf("texts %q", tt.Texts())
	}
	return tt
}

func TestKeyboard(t *testing.T) {
	tt := newTester(t)

	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyDown)
	if chosen != 2 {
		t.Errorf("after Down Down: chosen %d, want 2", chosen)
	}
	tt.Key(0, ui.KeyDown) // clamped at the end
	tt.Key(0, ui.KeyUp)
	if chosen != 1 {
		t.Errorf("after Down Down Down Up: chosen %d, want 1", chosen)
	}
	if !tt.Focused("Search repositories") {
		t.Error("search field lost the focus")
	}

	tt.Key(ui.Cmd, ui.KeyComma)
	if !showSettings || !tt.HasText("Save & Re-scan") {
		t.Fatal("Cmd+, did not open the settings")
	}
	tt.Key(0, ui.KeyUp) // inert behind the dialog
	if chosen != 1 {
		t.Errorf("Up behind the dialog moved the selection to %d", chosen)
	}
	tt.Key(0, ui.KeyEscape) // closes the dialog, not the panel
	if showSettings {
		t.Error("Escape left the settings open")
	}

	tt.Type("gam")
	if got := eng.GetState().FilteredRepos; len(got) != 1 || got[0].Name != "gamma" || chosen != 0 {
		t.Errorf("after typing: %v, chosen %d", got, chosen)
	}
	tt.Key(0, ui.KeyEscape) // clears the query before the panel hides
	if query != "" || len(eng.GetState().FilteredRepos) != 3 {
		t.Errorf("Escape with a query: query %q, %d repos", query, len(eng.GetState().FilteredRepos))
	}
}

// TestStreaming checks that repos the scanner adds above the selection
// leave it on the top row, the newest repo, as the list fills in.
func TestStreaming(t *testing.T) {
	tt := newTester(t)
	eng.AddRepos([]engine.Repo{{Name: "newest", Path: "/src/newest", LastCommitTime: now.Add(time.Minute)}})
	tt.Frame()
	if chosen != 0 || !tt.HasText("newest") {
		t.Errorf("chosen %d, texts %q", chosen, tt.Texts())
	}
}

// TestFocusAfterMouse checks that typing still searches after the mouse
// used a button or a row.
func TestFocusAfterMouse(t *testing.T) {
	tt := newTester(t)
	if err := tt.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Type("b")
	if query != "b" {
		t.Errorf("after the settings button: query %q", query)
	}

	tt = newTester(t)
	if err := tt.Click("beta"); err != nil {
		t.Fatal(err)
	}
	if chosen != 1 {
		t.Errorf("click on beta: chosen %d", chosen)
	}
	tt.Type("g")
	if query != "g" {
		t.Errorf("after a row click: query %q", query)
	}
	tt.Advance(time.Minute)
	if tt.FramePending() {
		t.Error("the idle view keeps asking for frames")
	}
}

func TestActionButtons(t *testing.T) {
	tt := newTester(t)
	var opened []string
	openRepo = func(path string, _ func(string) error) { opened = append(opened, path) }
	if err := tt.Click("Terminal"); err != nil { // on the chosen row, alpha
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "/src/alpha" {
		t.Errorf("opened %v", opened)
	}
	tt.Key(0, ui.KeyEnter)
	if len(opened) != 2 {
		t.Errorf("Enter: opened %v", opened)
	}
}
