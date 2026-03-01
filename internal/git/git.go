package git

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"time"
)

// GetRepoInfo extracts and returns the LastCommitMsg, LastCommitTime, and Branch
// for a given git repository path by invoking the git CLI.
func GetRepoInfo(path string) (msg string, t time.Time, branch string, err error) {
	// git log -1 --format="%an, %ar: %s" could be used, but we want structured data:
	// %s is subject
	// %cI is committer date, strict ISO 8601 format
	cmd := exec.Command("git", "-C", path, "log", "-1", "--format=%s|%cI")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Printf("git log failed for %s: %v", path, err)
		return "", time.Time{}, "", err
	}

	parts := strings.SplitN(strings.TrimSpace(out.String()), "|", 2)
	if len(parts) == 2 {
		msg = parts[0]
		parsedTime, err := time.Parse(time.RFC3339, parts[1])
		if err == nil {
			t = parsedTime
		}
	} else if len(parts) == 1 {
		msg = parts[0]
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
