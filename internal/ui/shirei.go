package ui

import (
	"os"
	"time"

	"github.com/cli/browser"

	"github.com/pteich/repo-hopper/internal/actions"
	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"

	"go.hasen.dev/shirei/app"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

var eng *engine.Engine
var settings *config.Settings

// OnSettingsChanged is called when the user saves new settings in the GUI.
var OnSettingsChanged func(s config.Settings)

// OnUpdateCheckRequested is called when the user manually checks for updates.
var OnUpdateCheckRequested func()

// Run initializes the Shirei application window and starts the rendering loop.
func Run(e *engine.Engine, s *config.Settings) {
	eng = e
	settings = s
	lastInteractionAt = time.Now()
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
var showUpdateDialog bool
var lastInteractionAt time.Time

const idleFocusBlurAfter = 10 * time.Second

// OpenURL opens a URL in the default browser using the OS default handler.
func OpenURL(url string) {
	_ = browser.OpenURL(url)
}

// fullWidthTextInput renders a TextInput that fills its container's width.
// TextInput defaults to ~10em; we set MinWidth from the enclosing container's
// resolved width (last frame's, so it self-corrects on resize). Call inside a
// container that establishes the desired width.
func fullWidthTextInput(buf *string) {
	inputAttrs := DefaultTextInputAttrs()
	if w := GetResolvedSize()[0]; w > 0 {
		inputAttrs.MinWidth = w
	}
	TextInputExt(buf, inputAttrs)
}

// RootView is called every frame by Shirei.
func RootView() {
	defer DebugPanel(true)
	defer PopupsHost()

	state := eng.GetState()

	if !hasAutoFocused {
		hasAutoFocused = true
	}

	maybeBlurIdleFocus()

	// Handle Global Keyboard Shortcuts
	handleKeyboardInput()

	// Outer layout filling the window
	// Extrinsic makes it accept the dimensions forced by the window system
	Container(Attrs(Extrinsic, Expand, Grow(1), Background(10, 10, 10, 1)), func() {

		// Search Bar Area
		Container(Attrs(Expand, MaxHeight(60), Pad(10), Background(0, 0, 15, 1), BorderColor(0, 0, 0, 1), BorderWidth(1)), func() {
			Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
				Icon(SymSearch)

				Container(Attrs(Expand, Grow(1)), func() {
					fullWidthTextInput(&searchQuery)
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
					Label("Scanning...", FontSize(12), TextColor(0, 0, 60, 1))
				} else {
					// Spacer to keep buttons in same position
					Element(Attrs(FixWidth(20)))
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

				// Check for Updates Icon
				if ButtonExt("", ButtonAttrs{
					Icon:     SymDownload,
					TextSize: 14,
				}) {
					showUpdateDialog = true
					go func() {
						if OnUpdateCheckRequested != nil {
							OnUpdateCheckRequested()
						}
					}()
				}
			})
		})

		// Results List Area
		Container(Attrs(Expand, Grow(1)), func() {
			if len(state.FilteredRepos) == 0 {
				Container(Attrs(Expand, Center, CrossMid), func() {
					if state.IsScanning {
						Label("Scanning for repositories...", FontSize(16), TextColor(0, 0, 50, 1))
					} else {
						Label("No repositories found.", FontSize(16), TextColor(0, 0, 50, 1))
					}
				})
				return
			}

			// Render VirtualListView using our custom scrolling logic
			ContainerWithKey("results-viewport", Attrs(Viewport, Expand), func() {
				const itemHeight = float32(60.0)

				viewRow := func(i int, width float32) {
					repo := state.FilteredRepos[i]
					isSelected := i == state.SelectedIndex

					// Row Styling
					rowAttrs := Attrs(Row, Expand, CrossMid, FixHeight(itemHeight), Pad2(10, 20), BorderColor(0, 0, 0, 1), BorderWidth(1))

					Container(rowAttrs, func() {
						if IsClicked() {
							eng.SetSelectedIndex(i)
							RequestNextFrame()
						}

						if isSelected {
							ModAttrs(Background(210, 80, 50, 1)) // Blue-ish highlight
						} else if IsHovered() {
							ModAttrs(Background(0, 0, 20, 1)) // Dark grey hover
						}

						// Color Selection for Text (White on Blue, Grey otherwise)
						nameColor := TextColor(0, 0, 90, 1)
						pathColor := TextColor(0, 0, 50, 1)
						subTextColor := TextColor(0, 0, 40, 1)
						branchColor := TextColor(0, 0, 60, 1)
						iconColor := TextColor(200, 40, 70, 1)

						if isSelected {
							nameColor = TextColor(0, 0, 100, 1)
							pathColor = TextColor(0, 0, 90, 1)
							subTextColor = TextColor(0, 0, 85, 1)
							branchColor = TextColor(0, 0, 100, 1)
							iconColor = TextColor(0, 0, 100, 1)
						}

						// Main row structure - clean and flat
						// Repo Icon
						Icon(SymFolder, FontSize(24), iconColor)
						Element(Attrs(FixWidth(16)))

						// Text Column
						Container(Attrs(Expand, Grow(1), Spacing(4)), func() {
							// Top Row: Name and Branch
							Container(Attrs(Row, CrossMid, Gap(8)), func() {
								Label(repo.Name, FontWeight(WeightBold), FontSize(16), nameColor)

								Container(Attrs(Background(0, 0, 10, 0.5), Corners(4), Pad2(2, 6)), func() {
									Label(repo.Branch, FontSize(12), branchColor)
								})

								// Show cached age
								Filler(1)
								Label(repo.CachedAge, FontSize(12), subTextColor)
							})

							// Bottom Row: Path or Commit Msg
							Label(repo.Path, FontSize(12), pathColor, Fonts(Monospace...))
						})

						// Shortcut Hints (only on selected row)
						if isSelected {
							Container(Attrs(Row, CrossMid, Gap(8)), func() {
								hintAttrs := Attrs(Row, CrossMid, Gap(4), Background(0, 0, 10, 0.3), Pad2(4, 8), Corners(4))

								// Editor Shortcut
								Container(hintAttrs, func() {
									Label("↵", FontWeight(WeightBold), FontSize(12), TextColor(0, 0, 95, 1))
									Icon(SymCode, FontSize(12), TextColor(0, 0, 95, 1))
									Label("Editor", FontSize(12), TextColor(0, 0, 95, 1))
								})

								// Terminal Shortcut
								Container(hintAttrs, func() {
									Label("⌘T", FontWeight(WeightBold), FontSize(12), TextColor(0, 0, 95, 1))
									Icon(SymExternal, FontSize(12), TextColor(0, 0, 95, 1))
									Label("Terminal", FontSize(12), TextColor(0, 0, 95, 1))
								})

								// Finder Shortcut
								Container(hintAttrs, func() {
									Label("⌘O", FontWeight(WeightBold), FontSize(12), TextColor(0, 0, 95, 1))
									Icon(SymFolder, FontSize(12), TextColor(0, 0, 95, 1))
									Label("Finder", FontSize(12), TextColor(0, 0, 95, 1))
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

	if showUpdateDialog {
		UpdateDialog()
	}
}

func maybeBlurIdleFocus() {
	if FrameInput.Mouse != 0 ||
		FrameInput.Motion != (Vec2{}) ||
		FrameInput.Scroll != (Vec2{}) ||
		FrameInput.Key != 0 ||
		FrameInput.Text != "" {
		lastInteractionAt = time.Now()
		return
	}

	// TextInput requests a redraw while focused for caret blinking.
	// If the user is idle, dropping focus lets the app fall back to low idle FPS.
	if !IdHasFocus(nil) && time.Since(lastInteractionAt) > idleFocusBlurAfter {
		FocusImmediateOn(nil)
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

// OnSkipUpdateVersion is called when user chooses to skip a specific version.
var OnSkipUpdateVersion func(version string)
