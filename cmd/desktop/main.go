package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
	"github.com/pteich/repo-hopper/internal/scanner"
	"github.com/pteich/repo-hopper/internal/ui"
	"github.com/pteich/repo-hopper/internal/updater"
)

// Build-time version information (set via ldflags)
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	scanMu     sync.Mutex
	scanCancel context.CancelFunc
)

func main() {
	// 1. Load user settings (or defaults if first run)
	settings := config.Load()

	// 2. Initialize Engine
	eng := engine.New()

	// 3. Setup settings change listener
	ui.OnSettingsChanged = func(newSettings config.Settings) {
		settings = newSettings
		eng.ClearRepos()
		runScan(eng, settings)
	}

	// 4. Run initial scan based on settings
	runScan(eng, settings)

	// 5. Start periodic re-scan in background
	go periodicRescan(eng, settings)

	// 6. Setup update checker
	setupUpdateChecker(&settings)

	// 7. Start the UI (blocks the main thread)
	ui.Run(eng, &settings)
}

func runScan(eng *engine.Engine, settings config.Settings) {
	scanMu.Lock()
	if scanCancel != nil {
		scanCancel()
	}
	scanCtx, cancel := context.WithCancel(context.Background())
	scanCancel = cancel
	scanMu.Unlock()

	cfg := scanner.ScanConfig{
		RootDirs:        settings.ScanDirs,
		MaxDepth:        settings.MaxDepth,
		Workers:         8,
		ExcludePatterns: settings.ExcludePatterns,
	}
	sc := scanner.NewScanner(cfg, eng)
	go sc.Start(scanCtx)
}

func periodicRescan(eng *engine.Engine, settings config.Settings) {
	if settings.RescanIntervalMin <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(settings.RescanIntervalMin) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Reload settings in case they changed
		settings = config.Load()
		eng.ClearRepos()
		runScan(eng, settings)
	}
}

func setupUpdateChecker(settings *config.Settings) {
	// Setup manual update check callback
	ui.OnUpdateCheckRequested = func() {
		checkUpdates(settings)
	}

	// Setup skip update version callback
	ui.OnSkipUpdateVersion = func(version string) {
		settings.SkipUpdateVersion = version
		_ = config.Save(*settings)
	}

	// Check for updates on startup if enabled
	if settings.CheckForUpdates {
		configDir := config.ConfigDir()
		shouldCheck, err := updater.ShouldCheckForUpdates(configDir, settings.UpdateCheckHours)
		if err == nil && shouldCheck {
			go func() {
				checkUpdates(settings)
				_ = updater.SetLastCheckTime(configDir)
			}()
		}
	}
}

func checkUpdates(settings *config.Settings) {
	currentVersion := updater.EnsureVersionPrefix(version)
	updateInfo, err := updater.CheckForUpdates(currentVersion, settings.SkipUpdateVersion)
	if err != nil {
		log.Printf("Failed to check for updates: %v", err)
		return
	}

	if updateInfo != nil {
		updateInfo.LatestVersion = updater.EnsureVersionPrefix(updateInfo.LatestVersion)
		log.Printf("New version available: %s (current: %s)", updateInfo.LatestVersion, updateInfo.CurrentVersion)
		ui.ShowUpdateDialog(updateInfo)
	} else {
		log.Println("Already up to date")
	}
}
