package ui

import (
	"bytes"
	"image/png"
	"log"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/ui"
)

// Repo Hopper is a launcher: a panel floating over every app, summoned by a
// global shortcut, which hides when the user goes elsewhere. The app lives
// in the menu bar (the notification area on Windows) without a Dock icon.

// shortcut is the registered global shortcut, "" while there is none.
var shortcut string

func newPanel() *mygo.Window {
	panel := mygo.NewWindow(mygo.WindowOptions{
		Title:         "Repo Hopper",
		Panel:         true,
		Hidden:        true,
		Frameless:     true,
		Transparent:   true,
		Vibrancy:      mygo.VibrancyPopover,
		Width:         800,
		Height:        600,
		DisableResize: true,
		Content:       ui.View(view),
	})
	// Hide rather than close, on Escape, Cmd-W or Alt-F4, and when the user
	// goes elsewhere: it shows again at once, as it was. A quit closes every
	// window first and stops when one refuses, so let the close through then.
	quitting := false
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	panel.OnClose(func(e *mygo.CloseEvent) {
		if quitting {
			return
		}
		e.PreventDefault()
		panel.Hide()
	})
	panel.OnBlur(func() {
		if !pickingFolders { // the folder picker's sheet takes the focus too
			panel.Hide()
		}
	})
	return panel
}

// show shows the panel with an empty search on the display with the
// pointer, a third of the way down as Spotlight does.
func show() {
	d := mygo.Screen.DisplayNearestPoint(mygo.Screen.CursorScreenPoint())
	w, h := win.Size()
	win.SetPosition(d.WorkArea.X+(d.WorkArea.Width-w)/2, d.WorkArea.Y+(d.WorkArea.Height-h)/3)
	if query != "" {
		query = ""
		eng.UpdateSearch("")
	}
	chosen = 0
	list.ScrollTo(0, ui.Start)
	search.Focus()
	win.Show()
}

func toggle() {
	if win.IsVisible() {
		win.Hide()
	} else {
		show()
	}
}

// setShortcut makes acc the global shortcut in place of the current one,
// which stays when acc cannot be registered (taken by another app, say).
func setShortcut(acc string) error {
	if acc == shortcut {
		return nil
	}
	if err := mygo.GlobalShortcut.Register(acc, toggle); err != nil {
		return err
	}
	if shortcut != "" {
		mygo.GlobalShortcut.Unregister(shortcut)
	}
	shortcut = acc
	return nil
}

// shortcutKeys spells a shortcut as the platform does: ⌥⇧Space on macOS.
func shortcutKeys(acc string) string {
	if runtime.GOOS != "darwin" {
		return acc
	}
	return strings.NewReplacer("CmdOrCtrl+", "⌘", "Cmd+", "⌘", "Ctrl+", "⌃", "Alt+", "⌥", "Shift+", "⇧").Replace(acc)
}

func setUpTray() {
	tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "Repo Hopper"})
	if err != nil {
		log.Print("no tray icon: ", err)
		return
	}
	tray.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Open Repo Hopper", Click: func(*mygo.MenuItem, *mygo.Window) { show() }},
		{Label: "Settings…", Click: func(*mygo.MenuItem, *mygo.Window) {
			show()
			openSettings()
		}},
		updater.MenuItem(),
		mygo.Separator(),
		{Label: "Open at Login", Type: mygo.MenuItemCheckbox, Checked: mygo.App.OpenAtLogin(),
			Click: func(it *mygo.MenuItem, _ *mygo.Window) {
				if err := mygo.App.SetOpenAtLogin(it.Checked); err != nil {
					log.Print(err)
				}
			}},
		mygo.Separator(),
		{Role: mygo.RoleQuit},
	}))
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { tray.Destroy() })
}

// trayIcon draws the folder icon as a 16 pt PNG: black for macOS to tint
// as a template, blue elsewhere to show on light and dark taskbars alike.
func trayIcon() []byte {
	ink := ui.RGBA(79, 124, 255, 1)
	if runtime.GOOS == "darwin" {
		ink = ui.RGBA(0, 0, 0, 1)
	}
	img := ui.Render(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		ui.Column(c).Fill().Center().Children(func() {
			ui.Icon(c, folderIcon).FontSize(16).TextColor(ink)
		})
	}, 16, 16, 2)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
