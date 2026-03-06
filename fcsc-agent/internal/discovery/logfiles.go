package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FindLatestLogFile searches the given base paths for the most recently modified
// file matching the pattern "metrics.log*". Returns the full path or empty string.
func FindLatestLogFile(basePaths []string) string {
	var candidates []logCandidate

	for _, base := range basePaths {
		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible paths
			}
			if info.IsDir() {
				name := info.Name()
				// Skip hidden dirs and common non-relevant directories
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == "target" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(info.Name(), "metrics.log") {
				candidates = append(candidates, logCandidate{
					path:    path,
					modTime: info.ModTime(),
				})
			}
			return nil
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	return candidates[0].path
}

// FindAllLogDirs returns all unique directories containing metrics.log* files.
func FindAllLogDirs(basePaths []string) []string {
	dirSet := make(map[string]struct{})

	for _, base := range basePaths {
		filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == "target" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(info.Name(), "metrics.log") {
				dirSet[filepath.Dir(path)] = struct{}{}
			}
			return nil
		})
	}

	dirs := make([]string, 0, len(dirSet))
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// TradersGarageDirFromLogPath infers the traders-garage repo root from a log file path.
// e.g., /home/ross/traders-garage/logs/metrics.log.xxx -> /home/ross/traders-garage
func TradersGarageDirFromLogPath(logPath string) string {
	dir := filepath.Dir(logPath) // .../logs
	parent := filepath.Dir(dir)  // .../traders-garage
	if filepath.Base(dir) == "logs" {
		return parent
	}
	return ""
}

type logCandidate struct {
	path    string
	modTime time.Time
}
