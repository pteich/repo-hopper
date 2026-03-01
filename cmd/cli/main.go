package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pteich/repo-hopper/internal/config"
	"github.com/pteich/repo-hopper/internal/engine"
	"github.com/pteich/repo-hopper/internal/scanner"
)

func main() {
	// 1. Load user settings
	settings := config.Load()

	// 2. Initialize Engine
	eng := engine.New()

	// 3. Setup Scanner
	cfg := scanner.ScanConfig{
		RootDirs:        settings.ScanDirs,
		MaxDepth:        settings.MaxDepth,
		Workers:         8,
		ExcludePatterns: settings.ExcludePatterns,
	}

	sc := scanner.NewScanner(cfg, eng)

	// 4. Start Scanning (Blocking)
	fmt.Print("Scanning repositories... ")
	start := time.Now()
	sc.Start(context.Background())
	duration := time.Since(start)

	state := eng.GetState()
	fmt.Printf("done. Found %d repositories in %v.\n", len(state.AllRepos), duration.Round(time.Millisecond))

	if len(state.AllRepos) == 0 {
		fmt.Println("No repositories found in the configured directories.")
		return
	}

	// 5. Show Top 5
	engine.SortRepos(state.AllRepos)
	fmt.Println("\nLatest 5 repositories:")
	for i := 0; i < 5 && i < len(state.AllRepos); i++ {
		r := state.AllRepos[i]
		fmt.Printf("%d. %s [%s] (%s)\n   %s\n",
			i+1, r.Name, r.Branch, r.LastCommitTime.Format("2006-01-02 15:04"), r.Path)
	}
}
