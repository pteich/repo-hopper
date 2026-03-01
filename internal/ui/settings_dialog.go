package ui

import (
	"log"
	"strconv"
	"strings"

	"github.com/pteich/repo-hopper/internal/config"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/tw"
	. "go.hasen.dev/shirei/widgets"
)

// Internal state for the settings dialog text fields
var (
	scanDirsStr   string
	editorStr     string
	terminalStr   string
	rescanStr     string
	maxDepthStr   string
	excludeStr    string
	initialized   bool
	validationErr string
)

func SettingsDialog() {
	if !initialized {
		scanDirsStr = strings.Join(settings.ScanDirs, ", ")
		editorStr = settings.Editor
		terminalStr = settings.Terminal
		rescanStr = strconv.Itoa(settings.RescanIntervalMin)
		maxDepthStr = strconv.Itoa(settings.MaxDepth)
		excludeStr = strings.Join(settings.ExcludePatterns, ", ")
		initialized = true
		validationErr = ""
	}

	Popup(func() {
		// Dark overlay - we use WindowSize directly to ensure it covers the whole screen
		// and Float(0,0) to anchor it to top-left regardless of where was called.
		overlayAttrs := TW(
			Float(0, 0),
			FixSizeV(WindowSize),
			Center,   // Center children horizontally
			CrossMid, // Center children vertically
			BG(0, 0, 0, 0.7),
		)

		Layout(overlayAttrs, func() {
			// Modal panel - Let it auto-size completely based on its children!
			modalAttrs := TW(
				FixWidth(500),
				BG(0, 0, 15, 1),
				BR(8),
				Pad(20),
				Spacing(15),
				Bo(0, 0, 30, 1),
				BW(1),
				Shd(20),
			)

			Layout(modalAttrs, func() {
				Label("Settings", FontWeight(WeightBold), Sz(20), Clr(0, 0, 100, 1))

				// Form Fields
				formField := func(label string, buf *string, hint string) {
					Layout(TW(Expand, Spacing(5)), func() {
						Label(label, Sz(14), Clr(0, 0, 70, 1))
						TextInput(buf)
						if hint != "" {
							Label(hint, Sz(11), Clr(0, 0, 40, 1))
						}
					})
				}

				formField("Scan Directories (comma separated)", &scanDirsStr, "e.g. /Users/peter/github, /tmp/repos")
				formField("Editor Command", &editorStr, "e.g. code, goland, nvim")
				formField("Terminal Command", &terminalStr, "e.g. Terminal, iTerm, Warp")

				Layout(TW(Row, Expand, Gap(20)), func() {
					formField("Re-scan Interval (min)", &rescanStr, "")
					formField("Max Scan Depth", &maxDepthStr, "")
				})

				formField("Exclude Patterns (comma separated)", &excludeStr, "e.g. node_modules, vendor, .cache")

				// Validation error display
				if validationErr != "" {
					Label(validationErr, Sz(12), Clr(80, 0, 0, 1))
				}

				// Buttons
				Layout(TW(Row, Expand, CrossMid, Gap(10)), func() {
					Filler(1)

					if ButtonExt("Cancel", ButtonAttrs{}) {
						showSettings = false
						initialized = false
						validationErr = ""
					}

					if ButtonExt("Save & Re-scan", ButtonAttrs{Primary: true}) {
						// Validate and parse integers
						rescanVal, err := strconv.Atoi(rescanStr)
						if err != nil || rescanVal < 0 {
							validationErr = "Invalid re-scan interval. Must be a non-negative integer."
							log.Printf("Invalid rescan interval: %s", rescanStr)
							return
						}

						depthVal, err := strconv.Atoi(maxDepthStr)
						if err != nil || depthVal <= 0 {
							validationErr = "Invalid max depth. Must be a positive integer."
							log.Printf("Invalid max depth: %s", maxDepthStr)
							return
						}

						// Validate scan directories
						scanDirs := splitAndTrim(scanDirsStr)
						if len(scanDirs) == 0 {
							validationErr = "At least one scan directory is required."
							return
						}

						// All validations passed
						settings.ScanDirs = scanDirs
						settings.Editor = editorStr
						settings.Terminal = terminalStr
						settings.RescanIntervalMin = rescanVal
						settings.MaxDepth = depthVal
						settings.ExcludePatterns = splitAndTrim(excludeStr)

						_ = config.Save(*settings)

						if OnSettingsChanged != nil {
							OnSettingsChanged(*settings)
						}

						showSettings = false
						initialized = false
						validationErr = ""
					}
				})
			})
		})
	})
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var res []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			res = append(res, t)
		}
	}
	return res
}
