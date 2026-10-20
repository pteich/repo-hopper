package ui

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/pteich/repo-hopper/internal/config"
)

// settingsForm holds the settings dialog's text fields while it is open.
var settingsForm struct {
	scanDirs, editor, terminal, rescan, maxDepth, exclude, shortcut string
	err                                                             string
}

// openSettings fills the form from the current settings and shows it.
func openSettings() {
	settingsForm.scanDirs = strings.Join(settings.ScanDirs, ", ")
	settingsForm.editor = settings.Editor
	settingsForm.terminal = settings.Terminal
	settingsForm.rescan = strconv.Itoa(settings.RescanIntervalMin)
	settingsForm.maxDepth = strconv.Itoa(settings.MaxDepth)
	settingsForm.exclude = strings.Join(settings.ExcludePatterns, ", ")
	settingsForm.shortcut = settings.GlobalShortcut
	settingsForm.err = ""
	showSettings = true
}

func settingsDialog(c *ui.Context) {
	f := &settingsForm
	ui.Column(c).Width(480).Gap(14).Children(func() {
		ui.Text(c, "Settings").FontSize(20).Bold()

		ui.Field(c, "Scan Directories (comma separated)", func() {
			ui.TextInput(c, &f.scanDirs).AutoFocus()
		}).Description("e.g. /Users/peter/github, /tmp/repos")
		ui.Row(c).Gap(20).Children(func() {
			ui.Field(c, "Editor Command", func() { ui.TextInput(c, &f.editor) }).
				Description("e.g. code, goland, nvim").Grow(1).Basis(0)
			ui.Field(c, "Terminal Command", func() { ui.TextInput(c, &f.terminal) }).
				Description("e.g. Terminal, iTerm, Warp").Grow(1).Basis(0)
		})
		ui.Row(c).Gap(20).Children(func() {
			ui.Field(c, "Re-scan Interval (min)", func() { ui.TextInput(c, &f.rescan) }).Grow(1).Basis(0)
			ui.Field(c, "Max Scan Depth", func() { ui.TextInput(c, &f.maxDepth) }).Grow(1).Basis(0)
		})
		ui.Field(c, "Exclude Patterns (comma separated)", func() { ui.TextInput(c, &f.exclude) }).
			Description("e.g. node_modules, vendor, .cache")
		ui.Field(c, "Global Shortcut", func() { ui.TextInput(c, &f.shortcut) }).
			Description("Shows and hides Repo Hopper from any app, e.g. Alt+Shift+Space, CmdOrCtrl+Shift+R")

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
	scanDirs := splitAndTrim(f.scanDirs)
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
	settings.ExcludePatterns = splitAndTrim(f.exclude)
	settings.GlobalShortcut = acc

	if err := config.Save(*settings); err != nil {
		log.Printf("saving settings: %v", err)
	}
	if OnSettingsChanged != nil {
		OnSettingsChanged(*settings)
	}
	showSettings = false
}

func splitAndTrim(s string) []string {
	var res []string
	for p := range strings.SplitSeq(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			res = append(res, t)
		}
	}
	return res
}
