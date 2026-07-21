package ui

import (
	"fmt"

	"github.com/pteich/repo-hopper/internal/updater"

	. "go.hasen.dev/shirei"
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
		overlayAttrs := Attrs(
			Float(0, 0),
			FixSizeVec(WindowSize),
			Center,
			CrossMid,
			Background(0, 0, 0, 0.7),
		)

		Container(overlayAttrs, func() {
			// Modal panel
			modalAttrs := Attrs(
				FixWidth(450),
				Background(0, 0, 15, 1),
				Corners(12),
				Pad(24),
				Spacing(16),
				BorderColor(0, 0, 30, 1),
				BorderWidth(1),
				BoxShadow(20),
			)

			Container(modalAttrs, func() {
				// Header with icon
				Container(Attrs(Row, Expand, Gap(12), CrossMid), func() {
					Label("🔄", FontSize(28))
					Container(Attrs(Expand), func() {
						Label("New Version Available!", FontWeight(WeightBold), FontSize(18), TextColor(0, 0, 100, 1))
						Label(fmt.Sprintf("%s → %s", updateInfo.CurrentVersion, updateInfo.LatestVersion),
							FontSize(14), TextColor(0, 0, 70, 1))
					})
				})

				// Divider
				Container(Attrs(Expand, MaxHeight(1)), func() {
					Element(Attrs(Expand, Background(0, 0, 30, 0.3)))
				})

				// Message
				Label("A new version of RepoHopper is available for download.",
					FontSize(13), TextColor(0, 0, 80, 1))

				// Release notes preview (if available)
				if updateInfo.ReleaseNotes != "" && len(updateInfo.ReleaseNotes) > 0 {
					Container(Attrs(Expand, Spacing(8)), func() {
						Label("What's New:", FontWeight(WeightBold), FontSize(12), TextColor(0, 0, 70, 1))

						// Show first few lines of release notes
						notes := updateInfo.ReleaseNotes
						if len(notes) > 200 {
							notes = notes[:200] + "..."
						}
						Label(notes, FontSize(11), TextColor(0, 0, 60, 1))
					})
				}

				// Buttons
				Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
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
					if ButtonExt("Download Now", ButtonAttrs{Accent: Vec4{210, 80, 50, 1}}) {
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
