package git

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetRepoInfo extracts and returns the LastCommitMsg, LastCommitTime, and Branch
// for a given git repository path by invoking the git CLI.
func GetRepoInfo(path string) (msg string, t time.Time, branch string, err error) {
	// git log -1 --format="%an, %ar: %s" could be used, but we want structured data:
	// %s is subject
	// %cI is committer date, strict ISO 8601 format
	// The date goes first: subjects may contain "|".
	cmd := exec.Command("git", "-C", path, "log", "-1", "--format=%cI|%s")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Printf("git log failed for %s: %v", path, err)
		return "", time.Time{}, "", err
	}

	parts := strings.SplitN(strings.TrimSpace(out.String()), "|", 2)
	if parsedTime, err := time.Parse(time.RFC3339, parts[0]); err == nil {
		t = parsedTime
	}
	if len(parts) == 2 {
		msg = parts[1]
	}

	// Get current branch
	cmdBranch := exec.Command("git", "-C", path, "branch", "--show-current")
	var outBranch bytes.Buffer
	cmdBranch.Stdout = &outBranch
	if err := cmdBranch.Run(); err == nil {
		branch = strings.TrimSpace(outBranch.String())
	} else {
		// Fallback for detached heads or older git versions
		cmdFallback := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
		var outFallback bytes.Buffer
		cmdFallback.Stdout = &outFallback
		if err := cmdFallback.Run(); err == nil {
			branch = strings.TrimSpace(outFallback.String())
		} else {
			log.Printf("git branch failed for %s: %v", path, err)
		}
	}

	return msg, t, branch, nil
}

// GetDirtyState reports whether the worktree at path has uncommitted changes
// (modified or untracked files), and returns the newest modification time
// among those files. The time is the zero time when the worktree is clean.
// Git stores no "dirty date", so file mtimes are the closest proxy; files
// listed but missing on disk (deleted, staged) are skipped.
func GetDirtyState(path string) (dirty bool, t time.Time, err error) {
	out, err := exec.Command("git", "-C", path,
		"ls-files", "-m", "-o", "--exclude-standard").Output()
	if err != nil {
		log.Printf("git ls-files failed for %s: %v", path, err)
		return false, time.Time{}, err
	}

	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f == "" {
			continue
		}
		dirty = true
		if fi, err := os.Stat(filepath.Join(path, f)); err == nil {
			if fi.ModTime().After(t) {
				t = fi.ModTime()
			}
		}
	}
	return dirty, t, nil
}
