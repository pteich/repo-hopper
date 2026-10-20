package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/pteich/repo-hopper/internal/config"
)

// settingsForm holds the settings dialog's text fields while it is open.
var settingsForm struct {
	scanDirs, exclude                            []string
	editor, terminal, rescan, maxDepth, shortcut string
	err                                          string
}

// pickingFolders is true while the folder picker shows, which takes the
// focus from the panel without the user going elsewhere.
var pickingFolders bool

// excludeSuggestions are directory names commonly left out of scans.
var excludeSuggestions = []string{"node_modules", "vendor", ".cache", "Library", "dist", "build", "target", ".venv", "Pods"}

// openSettings fills the form from the current settings and shows it.
func openSettings() {
	settingsForm.scanDirs = slices.Clone(settings.ScanDirs)
	settingsForm.editor = settings.Editor
	settingsForm.terminal = settings.Terminal
	settingsForm.rescan = strconv.Itoa(settings.RescanIntervalMin)
	settingsForm.maxDepth = strconv.Itoa(settings.MaxDepth)
	settingsForm.exclude = slices.Clone(settings.ExcludePatterns)
	settingsForm.shortcut = settings.GlobalShortcut
	settingsForm.err = ""
	showSettings = true
}

func settingsDialog(c *ui.Context) {
	f := &settingsForm
	ui.Column(c).Width(480).Gap(14).Children(func() {
		ui.Text(c, "Settings").FontSize(20).Bold()

		ui.Field(c, "Scan Directories", func() { scanDirsList(c) })
		ui.Row(c).Gap(20).Children(func() {
			ui.Field(c, "Editor Command", func() {
				ui.TextInput(c, &f.editor).Placeholder("e.g. code, goland, nvim")
			}).Grow(1).Basis(0)
			ui.Field(c, "Terminal Command", func() {
				ui.TextInput(c, &f.terminal).Placeholder("e.g. Terminal, iTerm, Warp")
			}).Grow(1).Basis(0)
		})
		ui.Row(c).Gap(20).Children(func() {
			ui.Field(c, "Re-scan Interval (min)", func() { ui.TextInput(c, &f.rescan) }).Grow(1).Basis(0)
			ui.Field(c, "Max Scan Depth", func() { ui.TextInput(c, &f.maxDepth) }).Grow(1).Basis(0)
		})
		ui.Field(c, "Exclude Directories", func() {
			ui.TokenField(c, &f.exclude, excludeSuggestions).Label("Exclude Directories").
				Placeholder("Add a name, then Enter")
		})
		ui.Field(c, "Show/Hide Shortcut (from any app)", func() {
			ui.TextInput(c, &f.shortcut).Placeholder("e.g. Alt+Shift+Space")
		})

		if f.err != "" {
			ui.Text(c, f.err).FontSize(12).TextColor(c.Theme().Danger)
		}

		ui.Row(c).Gap(10).Justify(ui.End).Children(func() {
			if ui.Button(c, "Cancel").Clicked() {
				showSettings = false
			}
			if ui.PrimaryButton(c, "Save & Re-scan").Clicked() {
				saveSettings()
			}
		})
	})
}

// scanDirsList lists the directories to scan, each with a button taking it
// out, and a button adding folders chosen in the system's folder picker.
func scanDirsList(c *ui.Context) {
	f := &settingsForm
	t := c.Theme()
	ui.Column(c).Gap(8).Children(func() {
		ui.List(c, nil, len(f.scanDirs), func(i int) {
			dir := f.scanDirs[i]
			ui.Row(c).Padding(4, 4, 4, 10).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, folderIcon).FontSize(14).TextColor(t.TextMuted)
				ui.Text(c, tildePath(dir)).Grow(1).SingleLine()
				// A callback, which runs after the build: removing the row
				// while the list builds its rows would index past the end.
				ui.Button(c, "").Label("Remove "+dir).Tooltip("Remove").Padding(2, 6).Children(func() {
					ui.Icon(c, removeIcon).FontSize(12)
				}).OnClick(func() {
					f.scanDirs = slices.DeleteFunc(f.scanDirs, func(d string) bool { return d == dir })
				})
			})
		}).MaxHeight(108).BorderWidth(1).BorderColor(t.Border).Radius(6).Padding(2).Children(func() {
			if len(f.scanDirs) == 0 {
				ui.Text(c, "No folders yet").TextColor(t.TextMuted).Padding(8, 10)
			}
		})
		if ui.Button(c, "Add Folders…").AlignSelf(ui.Start).Disabled(pickingFolders).Clicked() {
			pickFolders()
		}
	})
}

// pickFolders adds the folders the user chooses to the form. The picker
// blocks until it is answered, so it runs on a goroutine of its own.
func pickFolders() {
	pickingFolders = true
	home, _ := os.UserHomeDir()
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      win,
			Title:       "Add Folders to Scan",
			ButtonLabel: "Add",
			DefaultPath: home,
			Directory:   true,
			Multiple:    true,
		})
		win.Update(func() {
			pickingFolders = false
			if err != nil {
				settingsForm.err = "Cannot choose folders: " + err.Error()
				return
			}
			for _, p := range paths {
				if !slices.Contains(settingsForm.scanDirs, p) {
					settingsForm.scanDirs = append(settingsForm.scanDirs, p)
				}
			}
		})
	}()
}

// tildePath shortens a path in the home directory to ~/….
func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rest, ok := strings.CutPrefix(p, home); ok && (rest == "" || rest[0] == filepath.Separator) {
			return "~" + rest
		}
	}
	return p
}

// saveSettings validates the form, saves it and rescans; on an invalid
// field it keeps the dialog open with the error.
func saveSettings() {
	f := &settingsForm
	rescanVal, err := strconv.Atoi(strings.TrimSpace(f.rescan))
	if err != nil || rescanVal < 0 {
		f.err = "Invalid re-scan interval. Must be a non-negative integer."
		return
	}
	depthVal, err := strconv.Atoi(strings.TrimSpace(f.maxDepth))
	if err != nil || depthVal <= 0 {
		f.err = "Invalid max depth. Must be a positive integer."
		return
	}
	scanDirs := f.scanDirs
	if len(scanDirs) == 0 {
		f.err = "At least one scan directory is required."
		return
	}
	acc := strings.TrimSpace(f.shortcut)
	if acc == "" {
		f.err = "A global shortcut is required."
		return
	}
	if err := setShortcut(acc); err != nil {
		f.err = fmt.Sprintf("Cannot use %s: %v", acc, err)
		return
	}

	settings.ScanDirs = scanDirs
	settings.Editor = f.editor
	settings.Terminal = f.terminal
	settings.RescanIntervalMin = rescanVal
	settings.MaxDepth = depthVal
	settings.ExcludePatterns = slices.Clone(f.exclude)
	settings.GlobalShortcut = acc

	if err := config.Save(*settings); err != nil {
		log.Printf("saving settings: %v", err)
	}
	if OnSettingsChanged != nil {
		OnSettingsChanged(*settings)
	}
	showSettings = false
}
