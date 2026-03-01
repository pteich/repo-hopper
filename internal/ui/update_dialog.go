package ui

import (
	"fmt"

	"github.com/pteich/repo-hopper/internal/updater"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/tw"
	. "go.hasen.dev/shirei/widgets"
)

// UpdateInfo stores the latest update information
var updateInfo *updater.UpdateInfo

// ShowUpdateDialog displays a dialog when a new version is available
func ShowUpdateDialog(update *updater.UpdateInfo) {
	updateInfo = update
	showUpdateDialog = true
}

// UpdateDialog renders the update dialog UI
func UpdateDialog() {
	if updateInfo == nil {
		return
	}
	Popup(func() {
		// Dark overlay
		overlayAttrs := TW(
			Float(0, 0),
			FixSizeV(WindowSize),
			Center,
			CrossMid,
			BG(0, 0, 0, 0.7),
		)

		Layout(overlayAttrs, func() {
			// Modal panel
			modalAttrs := TW(
				FixWidth(450),
				BG(0, 0, 15, 1),
				BR(12),
				Pad(24),
				Spacing(16),
				Bo(0, 0, 30, 1),
				BW(1),
				Shd(20),
			)

			Layout(modalAttrs, func() {
				// Header with icon
				Layout(TW(Row, Expand, Gap(12), CrossMid), func() {
					Label("🔄", Sz(28))
					Layout(TW(Expand), func() {
						Label("New Version Available!", FontWeight(WeightBold), Sz(18), Clr(0, 0, 100, 1))
						Label(fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
							Sz(14), Clr(0, 0, 70, 1))
					})
				})

				// Divider
				Layout(TW(Expand, MaxHeight(1)), func() {
					Element(TW(Expand, BG(0, 0, 30, 0.3)))
				})

				// Message
				Label("A new version of RepoHopper is available for download.",
					Sz(13), Clr(0, 0, 80, 1))

				// Release notes preview (if available)
				if updateInfo.ReleaseNotes != "" && len(updateInfo.ReleaseNotes) > 0 {
					Layout(TW(Expand, Spacing(8)), func() {
						Label("What's New:", FontWeight(WeightBold), Sz(12), Clr(0, 0, 70, 1))

						// Show first few lines of release notes
						notes := updateInfo.ReleaseNotes
						if len(notes) > 200 {
							notes = notes[:200] + "..."
						}
						Label(notes, Sz(11), Clr(0, 0, 60, 1))
					})
				}

				// Buttons
				Layout(TW(Row, Expand, CrossMid, Gap(10)), func() {
					Filler(1)

					// Skip button
					if ButtonExt("Skip This Version", ButtonAttrs{}) {
						if OnSkipUpdateVersion != nil {
							OnSkipUpdateVersion(updateInfo.LatestVersion)
						}
						showUpdateDialog = false
						updateInfo = nil
					}

					// Remind Later button
					if ButtonExt("Remind Later", ButtonAttrs{}) {
						showUpdateDialog = false
						updateInfo = nil
					}

					// Download button
					if ButtonExt("Download Now", ButtonAttrs{Primary: true}) {
						showUpdateDialog = false
						// Open release URL in browser
						OpenURL(updateInfo.ReleaseURL)
						updateInfo = nil
					}
				})
			})
		})
	})
}

// CheckForUpdatesButton shows a button that triggers update check
func CheckForUpdatesButton() {
	if ButtonExt("Check for Updates", ButtonAttrs{}) {
		showUpdateDialog = true
		go func() {
			// Trigger update check - will show dialog if update found
			if OnUpdateCheckRequested != nil {
				OnUpdateCheckRequested()
			}
		}()
	}
}
