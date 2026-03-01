package actions

import (
	"os/exec"
	"runtime"
)

// OpenInTerminal attempts to open a new terminal window at the given path.
// terminalApp is the configured terminal program (e.g. "Terminal", "iTerm", "Warp").
func OpenInTerminal(terminalApp string, path string) error {
	if terminalApp == "" {
		terminalApp = "Terminal"
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-a", terminalApp, path).Start()
	case "windows":
		// TODO: use a more robust way to open a terminal at a given path
		return exec.Command("cmd", "/c", "start", terminalApp, "/K", "cd /d "+path).Start()
	case "linux":
		// TODO: use a more robust way to open a terminal at a given path
		return exec.Command(terminalApp, "--working-directory="+path).Start()
	default:
		return exec.Command("open", path).Start()
	}
}

// OpenInFinder attempts to open the default file manager at the given path.
func OpenInFinder(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "explorer"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, path)
	return exec.Command(cmd, args...).Start()
}

// OpenInEditor attempts to open the given pathway in the specified editor CLI (e.g. "code", "goland", "vim").
func OpenInEditor(editor string, path string) error {
	if editor == "" {
		// TODO Fallback to Code if not specified, could be environment var $EDITOR
		editor = "code"
	}
	return exec.Command(editor, path).Start()
}
