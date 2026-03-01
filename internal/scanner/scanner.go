package scanner

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pteich/repo-hopper/internal/engine"
	"github.com/pteich/repo-hopper/internal/git"
)

// ScanConfig holds configuration for the repository scanning process.
type ScanConfig struct {
	// RootDirs are the starting directories for the scan (e.g. []string{"/Users/peter/Development"})
	RootDirs []string

	// MaxDepth limits how deep the scanner will go to prevent infinite loops or scanning huge trees
	MaxDepth int

	// Workers is the number of concurrent goroutines extracting git metadata
	Workers int

	// ExcludePatterns are directory names to skip during scanning
	ExcludePatterns []string
}

// Scanner orchestrates the discovery of Git repositories.
type Scanner struct {
	config ScanConfig
	engine *engine.Engine
}

func NewScanner(cfg ScanConfig, eng *engine.Engine) *Scanner {
	if cfg.Workers <= 0 {
		cfg.Workers = 4 // Default sensible workers
	}
	return &Scanner{
		config: cfg,
		engine: eng,
	}
}

// Start initiates the scanning process in the background.
// It searches for .git directories, parses them concurrently, and updates the engine state.
// We pass a context to allow cancellation if needed.
func (s *Scanner) Start(ctx context.Context) {
	s.engine.SetScanning(true)
	defer s.engine.SetScanning(false)

	// Channel to send discovered git repository paths to workers
	repoPaths := make(chan string, 100)

	// Channel to collect parsed Engine Repo objects from workers
	parsedRepos := make(chan engine.Repo, 100)

	var wgWorkers sync.WaitGroup
	var wgCollector sync.WaitGroup

	// Start workers for git metadata extraction
	for i := 0; i < s.config.Workers; i++ {
		wgWorkers.Add(1)
		go func() {
			defer wgWorkers.Done()
			for path := range repoPaths {
				// Prevent work if context is done
				select {
				case <-ctx.Done():
					return
				default:
				}

				msg, t, branch, err := git.GetRepoInfo(path)
				if err != nil {
					log.Printf("Failed to get git info for %s: %v", path, err)
					continue
				}

				repo := engine.Repo{
					Path:           path,
					Name:           filepath.Base(path),
					LastCommitMsg:  msg,
					LastCommitTime: t,
					Branch:         branch,
				}
				repo.ComputeAge()
				parsedRepos <- repo
			}
		}()
	}

	// Start a collector to batch insert repos into the engine
	wgCollector.Add(1)
	go func() {
		defer wgCollector.Done()

		var batch []engine.Repo
		ticker := time.NewTicker(500 * time.Millisecond) // Update UI every 500ms
		defer ticker.Stop()

		for {
			select {
			case repo, ok := <-parsedRepos:
				if !ok {
					// Flush remaining
					if len(batch) > 0 {
						s.engine.AddRepos(batch)
					}
					return
				}
				batch = append(batch, repo)

				// If batch gets huge, flush immediately to avoid mem spikes
				if len(batch) >= 100 {
					s.engine.AddRepos(batch)
					batch = batch[:0]
				}
			case <-ticker.C:
				if len(batch) > 0 {
					s.engine.AddRepos(batch)
					batch = batch[:0] // Retain capacity
				}
			}
		}
	}()

	// Perform the actual filesystem walk
	for _, rootDir := range s.config.RootDirs {
		// Calculate depth to respect MaxDepth
		rootDepth := getDepth(rootDir)

		err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			// Check context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			if err != nil {
				return nil // Skip paths we don't have permission to read
			}

			if !d.IsDir() {
				return nil
			}

			// Respect MaxDepth
			if s.config.MaxDepth > 0 && getDepth(path)-rootDepth > s.config.MaxDepth {
				return filepath.SkipDir
			}

			// Check if this directory is named ".git"
			if d.Name() == ".git" {
				// We found a repo! The actual repo path is the parent directory.
				repoPath := filepath.Dir(path)

				// Send to workers
				repoPaths <- repoPath

				// IMPORTANT: Skip traversing inside the .git folder itself
				return filepath.SkipDir
			}

			// Optimization: Skip excluded directories from config
			for _, pattern := range s.config.ExcludePatterns {
				if d.Name() == pattern {
					return filepath.SkipDir
				}
			}

			// Also skip hidden directories that aren't .git itself
			if strings.HasPrefix(d.Name(), ".") && d.Name() != ".git" {
				return filepath.SkipDir
			}

			return nil
		})

		if err != nil {
			log.Printf("Error walking directory %s: %v", rootDir, err)
		}
	}

	// Close the jobs channel to signal workers to stop
	close(repoPaths)

	// Wait for all workers to finish their git parsing
	wgWorkers.Wait()

	// Close the results channel to signal collector to stop
	close(parsedRepos)

	// Wait for collector to finish flushing
	wgCollector.Wait()
}

// getDepth calculates the depth of a filepath
func getDepth(p string) int {
	return len(strings.Split(filepath.Clean(p), string(os.PathSeparator)))
}
