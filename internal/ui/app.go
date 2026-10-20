package ui

import (
	"fmt"
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"

	"github.com/pteich/repo-hopper/internal/actions"
	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
)

var (
	eng      *engine.Engine
	settings *config.Settings
	win      *mygo.Window
)

// OnReady is called on the main thread once the window exists. Start
// scanning here: the engine's OnChange redraws the window, so it must not
// fire before there is one.
var OnReady func()

// OnSettingsChanged is called when the user saves new settings in the GUI.
var OnSettingsChanged func(s config.Settings)

// View state. It is only read and written on the main thread: by the view,
// and by callbacks MyGo runs there (Window.Update, menus, the shortcut).
var (
	query        string
	chosen       int // index into the engine's FilteredRepos
	list         ui.ListState
	search       ui.Handle
	showSettings bool
)

// Run starts the launcher and blocks in MyGo's event loop until the app
// quits. A second launch shows the panel of the running one instead.
func Run(e *engine.Engine, s *config.Settings) {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	eng = e
	settings = s
	mygo.Use(native.Plugin) // the update window, and background update checks
	// No Dock icon or app switcher entry (macOS); packaged apps also say so
	// with LSUIElement in mygo.json, from the start.
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnWindowAllClosed(func() {}) // keep running while the panel hides
	mygo.App.OnSecondInstance(func([]string, string) { show() })
	mygo.App.WhenReady(func() {
		win = newPanel()
		// The scanner changes the engine from its goroutines; Invalidate is
		// safe from any goroutine and the view reads the engine under its lock.
		eng.OnChange = win.Invalidate
		if err := setShortcut(settings.GlobalShortcut); err != nil {
			log.Printf("global shortcut %s: %v", settings.GlobalShortcut, err)
		}
		setUpTray()
		if OnReady != nil {
			OnReady()
		}
		if !mygo.App.WasOpenedAtLogin() {
			show() // started by the user: show what it does
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// view builds the window for every frame.
func view(c *ui.Context) {
	t := c.Theme()
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent) // the panel's material shows through
	} else {
		c.Root().Background(t.Background)
	}

	// One snapshot per build, so the rows below index a list that cannot
	// change under them while the scanner adds repos.
	state := eng.GetState()
	repos := state.FilteredRepos
	chosen = max(0, min(chosen, len(repos)-1))

	handleKeyboardInput(c, repos)

	// Typing always searches: take the focus back from a clicked row or
	// button, or from a dialog once it closes.
	if !showSettings && !search.Focused(c) {
		search.Focus()
	}

	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Padding(10, 14).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, searchIcon).FontSize(18).TextColor(t.TextMuted)
			ui.TextInputBase(c, &query).Bind(&search).Grow(1).FontSize(18).
				Placeholder("Search repositories").Label("Search repositories").
				OnChange(func() {
					eng.UpdateSearch(query)
					chosen = 0
					list.ScrollTo(0, ui.Start)
				}).
				OnSubmit(func() { openChosen(eng.GetState().FilteredRepos, openInEditor) }) // the field takes Enter
			if iconButton(c, refreshIcon, "Rescan").Clicked() {
				rescan()
			}
			if iconButton(c, settingsIcon, "Settings").Clicked() {
				openSettings()
			}
			if iconButton(c, downloadIcon, "Check for Updates").Clicked() {
				updater.CheckForUpdates()
			}
		})
		ui.Divider(c)

		// No list.Key: rows hold no state, and a Key would make the selection
		// follow its repo down as the scanner adds newer ones above it.
		list.Selected = &chosen
		list.Label = func(i int) string { return repos[i].Name }
		results := ui.List(c, &list, len(repos), func(i int) { repoRow(c, repos[i], i == chosen) }).
			Grow(1).Padding(6).Children(func() {
			if len(repos) == 0 {
				msg := "No repositories found."
				if state.IsScanning {
					msg = "Scanning for repositories…"
				}
				ui.Text(c, msg).FontSize(16).TextColor(t.TextMuted).Padding(24).AlignSelf(ui.Center)
			}
		})
		if results.Submitted() { // double click, or Enter while the list has the focus
			openChosen(repos, openInEditor)
		}

		ui.Divider(c)
		statusBar(c, state)
	})

	ui.Modal(c, &showSettings, func() { settingsDialog(c) })
}

// repoRow shows a repo: name and branch over its path, its age, and buttons
// for the chosen or hovered row that open it.
func repoRow(c *ui.Context, repo engine.Repo, selected bool) {
	t := c.Theme()
	bg, text, muted := ui.Transparent, t.Text, t.TextMuted
	if selected {
		bg, text, muted = t.Accent, t.AccentText, t.AccentText.Alpha(0.8)
	}
	row := ui.Row(c).Padding(8, 14).Gap(14).AlignItems(ui.Center).Radius(6).Background(bg)
	hovered := row.Hovered()
	row.Children(func() {
		ui.Icon(c, folderIcon).FontSize(22).TextColor(muted)
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, repo.Name).Bold().FontSize(15).TextColor(text).SingleLine()
				if repo.Branch != "" {
					ui.Text(c, repo.Branch).FontSize(11).TextColor(text).SingleLine().
						Padding(1, 6).Radius(4).Background(text.Alpha(0.12))
				}
			})
			ui.Text(c, repo.Path).FontSize(12).Font("monospace").TextColor(muted).SingleLine()
		})
		ui.Text(c, repo.CachedAge).FontSize(12).TextColor(muted).SingleLine()
		if selected || hovered {
			ui.Row(c).Gap(6).Children(func() {
				for _, a := range repoActions {
					if actionButton(c, a.label, a.keys).Clicked() {
						openRepo(repo.Path, a.open)
					}
				}
			})
		}
	})
}

// repoActions are the ways to open a repo, as the row's buttons show them.
var repoActions = []struct {
	label, keys string
	open        func(path string) error
}{
	{"Editor", "↵", openInEditor},
	{"Terminal", cmdKey + "T", openInTerminal},
	{"Finder", cmdKey + "O", actions.OpenInFinder},
}

// cmdKey spells ui.Cmd as the platform does.
var cmdKey = map[bool]string{true: "⌘", false: "Ctrl+"}[runtime.GOOS == "darwin"]

func actionButton(c *ui.Context, label, keys string) ui.Element {
	return ui.Button(c, "").Label(label).Padding(3, 8).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(12).SingleLine()
		ui.Text(c, keys).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine()
	})
}

// statusBar shows scanning progress while a scan runs and the repo count
// otherwise; add more status text here as the app grows.
func statusBar(c *ui.Context, state engine.State) {
	t := c.Theme()
	msg := "Scanning for repositories…"
	if !state.IsScanning {
		msg = pluralRepos(len(state.FilteredRepos))
	}
	ui.Row(c).Padding(6, 14).Children(func() {
		ui.Text(c, msg).FontSize(12).TextColor(t.TextMuted).Grow(1)
		if shortcut != "" {
			ui.Text(c, shortcutKeys(shortcut)+" to show or hide").FontSize(12).TextColor(t.TextMuted)
		}
	})
}

func handleKeyboardInput(c *ui.Context, repos []engine.Repo) {
	// A dialog takes Escape first and closes, and the shortcuts below wait
	// while one is open. With a query, Escape clears it; else it is left to
	// the panel, which closes, and so hides.
	if query != "" && c.Shortcut(0, ui.KeyEscape) {
		query = ""
		eng.UpdateSearch("")
		chosen = 0
	}
	c.OnShortcut(ui.Cmd, ui.KeyQ, mygo.App.Quit)
	if c.Shortcut(0, ui.KeyDown) {
		choose(chosen + 1)
	}
	if c.Shortcut(0, ui.KeyUp) {
		choose(chosen - 1)
	}
	if c.Shortcut(0, ui.KeyEnter) || c.Shortcut(ui.Cmd, ui.KeyE) {
		openChosen(repos, openInEditor)
	}
	if c.Shortcut(ui.Cmd, ui.KeyT) {
		openChosen(repos, openInTerminal)
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) {
		openChosen(repos, actions.OpenInFinder)
	}
	if c.Shortcut(ui.Cmd, ui.KeyR) {
		rescan()
	}
	if c.Shortcut(ui.Cmd, ui.KeyComma) {
		openSettings()
	}
}

// choose moves the selection, clamped to the list, and scrolls it into view.
func choose(i int) {
	n := len(eng.GetState().FilteredRepos)
	if n == 0 {
		return
	}
	chosen = max(0, min(i, n-1))
	list.ScrollIntoView(chosen)
}

func openInEditor(path string) error   { return actions.OpenInEditor(settings.Editor, path) }
func openInTerminal(path string) error { return actions.OpenInTerminal(settings.Terminal, path) }

func openChosen(repos []engine.Repo, how func(path string) error) {
	if chosen >= 0 && chosen < len(repos) {
		openRepo(repos[chosen].Path, how)
	}
}

// openRepo hides the panel, which gives the keyboard back to the app the
// user was in before the opened app takes it. It is replaced in tests.
var openRepo = func(path string, how func(path string) error) {
	win.Hide()
	if err := how(path); err != nil {
		log.Printf("opening %s: %v", path, err)
	}
}

func rescan() {
	if OnSettingsChanged != nil {
		OnSettingsChanged(*settings)
	}
}

func iconButton(c *ui.Context, icon *ui.SVG, label string) ui.Element {
	return ui.Button(c, "").Label(label).Tooltip(label).Children(func() {
		ui.Icon(c, icon).FontSize(15)
	})
}

func pluralRepos(n int) string {
	if n == 1 {
		return "1 repository"
	}
	return fmt.Sprintf("%d repositories", n)
}
