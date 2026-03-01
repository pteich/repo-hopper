package main

import (
	"context"
	"time"

	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
	"github.com/pteich/repo-hopper/internal/scanner"
	"github.com/pteich/repo-hopper/internal/ui"
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

	// 6. Start the UI (blocks the main thread)
	ui.Run(eng, &settings)
}

func runScan(eng *engine.Engine, settings config.Settings) {
	cfg := scanner.ScanConfig{
		RootDirs:        settings.ScanDirs,
		MaxDepth:        settings.MaxDepth,
		Workers:         8,
		ExcludePatterns: settings.ExcludePatterns,
	}
	sc := scanner.NewScanner(cfg, eng)
	go sc.Start(context.Background())
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
