package discovery

import (
	"os/exec"
	"strings"
)

// GitBranch returns the current git branch for the given repo directory.
// Returns "unknown" if detection fails.
func GitBranch(repoDir string) string {
	if repoDir == "" {
		return "unknown"
	}

	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "unknown"
	}
	return branch
}
