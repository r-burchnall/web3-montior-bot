package checks

import (
	"os"
	"path/filepath"
	"strings"
)

// BotRunning checks if a process matching the given name is running.
// If processName is empty, it looks for any process whose cmdline contains "traders-garage" or "arb".
func BotRunning(processName string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Only look at numeric directories (PIDs)
		if len(entry.Name()) == 0 || entry.Name()[0] < '0' || entry.Name()[0] > '9' {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}

		cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
		if processName != "" {
			if strings.Contains(cmd, processName) {
				return true
			}
		} else {
			// Auto-detect: look for common arb bot indicators
			lower := strings.ToLower(cmd)
			if strings.Contains(lower, "traders-garage") || strings.Contains(lower, "arb-bot") || strings.Contains(lower, "arb_bot") {
				return true
			}
		}
	}

	return false
}
