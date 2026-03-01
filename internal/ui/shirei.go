package ui

import (
	"os"
	"time"

	"github.com/pteich/repo-hopper/internal/actions"
	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"

	app "go.hasen.dev/shirei/giobackend"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/tw"
	. "go.hasen.dev/shirei/widgets"
)

var eng *engine.Engine
var settings *config.Settings

// OnSettingsChanged is called when the user saves new settings in the GUI.
var OnSettingsChanged func(s config.Settings)

// Run initializes the Shirei application window and starts the rendering loop.
func Run(e *engine.Engine, s *config.Settings) {
	eng = e
	settings = s
	app.SetupWindow("Repo Hopper", 800, 600)
	app.Run(RootView)
}

// Search debounce
var searchQuery string
var searchDebounceTimer *time.Timer
var pendingSearchQuery string

const searchDebounceMs = 150

var hasAutoFocused bool
var showSettings bool

// RootView is called every frame by Shirei.
func RootView() {
	defer DebugPanel(true)
	defer PopupsHost()

	state := eng.GetState()

	if !hasAutoFocused {
		hasAutoFocused = true
	}

	// Handle Global Keyboard Shortcuts
	handleKeyboardInput()

	// Outer layout filling the window
	// Extrinsic makes it accept the dimensions forced by the window system
	Layout(TW(Extrinsic, Expand, Grow(1), BG(10, 10, 10, 1)), func() {

		// Search Bar Area
		Layout(TW(Expand, MaxHeight(60), Pad(10), BG(0, 0, 15, 1), Bo(0, 0, 0, 1), BW(1)), func() {
			Layout(TW(Row, Expand, CrossMid, Gap(10)), func() {
				Icon(SymSearch)

				Layout(TW(Expand, Grow(1)), func() {
					TextInput(&searchQuery)
				})

				// Debounced search update
				// We track pendingSearchQuery to avoid resetting the timer on every
				// frame while waiting for it to fire (immediate-mode renders repeatedly
				// and the condition searchQuery != state.SearchQuery stays true until
				// the timer fires and updates the engine).
				if searchQuery != state.SearchQuery && searchQuery != pendingSearchQuery {
					// Reset timer on each actual keystroke
					if searchDebounceTimer != nil {
						searchDebounceTimer.Stop()
					}
					pendingSearchQuery = searchQuery
					searchDebounceTimer = time.AfterFunc(time.Duration(searchDebounceMs)*time.Millisecond, func() {
						eng.UpdateSearch(searchQuery)
						pendingSearchQuery = ""
						RequestNextFrame() // Trigger UI redraw after search results change
					})
				}

				// Scanning indicator - always reserve space to prevent button movement
				if state.IsScanning {
					Icon(SymClock)
					Label("Scanning...", Sz(12), Clr(0, 0, 60, 1))
				} else {
					// Spacer to keep buttons in same position
					Element(TW(FixWidth(20)))
				}

				// Refresh button
				if ButtonExt("", ButtonAttrs{
					Icon:     SymRefresh,
					TextSize: 14,
				}) {
					eng.ClearRepos()
					if OnSettingsChanged != nil {
						OnSettingsChanged(*settings)
					}
				}

				// Settings Gear Icon
				if ButtonExt("", ButtonAttrs{
					Icon:     SymCog,
					TextSize: 14,
				}) {
					showSettings = true
				}
			})
		})

		// Results List Area
		Layout(TW(Expand, Grow(1)), func() {
			if len(state.FilteredRepos) == 0 {
				Layout(TW(Expand, Center, CrossMid), func() {
					if state.IsScanning {
						Label("Scanning for repositories...", Sz(16), Clr(0, 0, 50, 1))
					} else {
						Label("No repositories found.", Sz(16), Clr(0, 0, 50, 1))
					}
				})
				return
			}

			// Render VirtualListView using our custom scrolling logic
			LayoutId("results-viewport", TW(Viewport, Expand), func() {
				const itemHeight = float32(60.0)

				viewRow := func(i int, width float32) {
					repo := state.FilteredRepos[i]
					isSelected := i == state.SelectedIndex

					// Row Styling
					rowAttrs := TW(Row, Expand, CrossMid, FixHeight(itemHeight), Pad2(10, 20), Bo(0, 0, 0, 1), BW(1))

					Layout(rowAttrs, func() {
						if isSelected {
							ModAttrs(BG(210, 80, 50, 1)) // Blue-ish highlight
						} else if IsHovered() {
							ModAttrs(BG(0, 0, 20, 1)) // Dark grey hover
						}

						// Color Selection for Text (White on Blue, Grey otherwise)
						nameColor := Clr(0, 0, 90, 1)
						pathColor := Clr(0, 0, 50, 1)
						subTextColor := Clr(0, 0, 40, 1)
						branchColor := Clr(0, 0, 60, 1)
						iconColor := Clr(200, 40, 70, 1)

						if isSelected {
							nameColor = Clr(0, 0, 100, 1)
							pathColor = Clr(0, 0, 90, 1)
							subTextColor = Clr(0, 0, 85, 1)
							branchColor = Clr(0, 0, 100, 1)
							iconColor = Clr(0, 0, 100, 1)
						}

						// Main row structure - clean and flat
						// Repo Icon
						Icon(SymFolder, Sz(24), iconColor)
						Element(TW(FixWidth(16)))

						// Text Column
						Layout(TW(Expand, Grow(1), Spacing(4)), func() {
							// Top Row: Name and Branch
							Layout(TW(Row, CrossMid, Gap(8)), func() {
								Label(repo.Name, FontWeight(WeightBold), Sz(16), nameColor)

								Layout(TW(BG(0, 0, 10, 0.5), BR(4), Pad2(2, 6)), func() {
									Label(repo.Branch, Sz(12), branchColor)
								})

								// Show cached age
								Filler(1)
								Label(repo.CachedAge, Sz(12), subTextColor)
							})

							// Bottom Row: Path or Commit Msg
							Label(repo.Path, Sz(12), pathColor, Fonts(Monospace...))
						})

						// Shortcut Hints (only on selected row)
						if isSelected {
							Layout(TW(Row, CrossMid, Gap(8)), func() {
								hintAttrs := TW(Row, CrossMid, Gap(4), BG(0, 0, 10, 0.3), Pad2(4, 8), BR(4))

								// Editor Shortcut
								Layout(hintAttrs, func() {
									Label("↵", FontWeight(WeightBold), Sz(12), Clr(0, 0, 95, 1))
									Icon(SymCode, Sz(12), Clr(0, 0, 95, 1))
									Label("Editor", Sz(12), Clr(0, 0, 95, 1))
								})

								// Terminal Shortcut
								Layout(hintAttrs, func() {
									Label("⌘T", FontWeight(WeightBold), Sz(12), Clr(0, 0, 95, 1))
									Icon(SymExternal, Sz(12), Clr(0, 0, 95, 1))
									Label("Terminal", Sz(12), Clr(0, 0, 95, 1))
								})

								// Finder Shortcut
								Layout(hintAttrs, func() {
									Label("⌘O", FontWeight(WeightBold), Sz(12), Clr(0, 0, 95, 1))
									Icon(SymFolder, Sz(12), Clr(0, 0, 95, 1))
									Label("Finder", Sz(12), Clr(0, 0, 95, 1))
								})
							})
						}
					})
				}

				entryId := func(index int) any {
					return state.FilteredRepos[index].Path
				}

				entryHeight := func(index int, width float32) float32 {
					return itemHeight
				}

				VirtualListWithScroll(len(state.FilteredRepos), entryId, entryHeight, viewRow, state.SelectedIndex)
			})
		})
	})

	if showSettings {
		SettingsDialog()
	}
}

func handleKeyboardInput() {
	state := eng.GetState()

	switch FrameInput.Key {
	case KeyEscape:
		if showSettings {
			showSettings = false
		} else {
			os.Exit(0) // Quit application on ESC
		}
	case KeyDown:
		eng.SelectNext()
		RequestNextFrame()
	case KeyUp:
		eng.SelectPrev()
		RequestNextFrame()
	case KeyEnter:
		if state.SelectedIndex >= 0 && state.SelectedIndex < len(state.FilteredRepos) {
			repo := state.FilteredRepos[state.SelectedIndex]
			_ = actions.OpenInEditor(settings.Editor, repo.Path)
		}
	}

	// Complex Modifiers (e.g. Cmd+E, Cmd+T, Cmd+O, Cmd+, Cmd+R)
	if InputState.Modifiers&ModSuper != 0 || InputState.Modifiers&ModCmd != 0 {
		switch FrameInput.Key {
		case KeyE:
			if state.SelectedIndex >= 0 && state.SelectedIndex < len(state.FilteredRepos) {
				repo := state.FilteredRepos[state.SelectedIndex]
				_ = actions.OpenInEditor(settings.Editor, repo.Path)
			}
		case KeyT:
			if state.SelectedIndex >= 0 && state.SelectedIndex < len(state.FilteredRepos) {
				repo := state.FilteredRepos[state.SelectedIndex]
				_ = actions.OpenInTerminal(settings.Terminal, repo.Path)
			}
		case KeyO:
			if state.SelectedIndex >= 0 && state.SelectedIndex < len(state.FilteredRepos) {
				repo := state.FilteredRepos[state.SelectedIndex]
				_ = actions.OpenInFinder(repo.Path)
			}
		case KeyR:
			// Manual refresh
			eng.ClearRepos()
			if OnSettingsChanged != nil {
				OnSettingsChanged(*settings)
			}
		case KeyCode(','):
			showSettings = true
		}
	}
}
