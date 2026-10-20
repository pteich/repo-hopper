package main

import (
	"context"
	"sync"
	"time"

	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
	"github.com/pteich/repo-hopper/internal/scanner"
	"github.com/pteich/repo-hopper/internal/ui"
)

var (
	scanMu     sync.Mutex
	scanCancel context.CancelFunc
)

func main() {
	settings := config.Load()

	eng := engine.New()

	ui.OnSettingsChanged = func(newSettings config.Settings) {
		settings = newSettings
		eng.ClearRepos()
		runScan(eng, settings)
	}

	ui.OnReady = func() {
		runScan(eng, settings)
		go periodicRescan(eng, settings)
	}

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
