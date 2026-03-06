package logparser

import (
	"bufio"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Tailer follows the latest log file and continuously parses new lines.
type Tailer struct {
	mu       sync.RWMutex
	state    *ParsedState
	filePath string
	stopCh   chan struct{}
}

func NewTailer() *Tailer {
	return &Tailer{
		state:  NewParsedState(),
		stopCh: make(chan struct{}),
	}
}

// State returns a snapshot of the current parsed state.
func (t *Tailer) State() *ParsedState {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a shallow copy of the state
	cp := &ParsedState{
		Queues:           make(map[string]*QueueStats, len(t.state.Queues)),
		LastGeyserUpdate: t.state.LastGeyserUpdate,
		LastLineTime:     t.state.LastLineTime,
	}
	for k, v := range t.state.Queues {
		stats := *v
		cp.Queues[k] = &stats
	}
	return cp
}

// SetFile updates which file the tailer is following.
// If the file changes, the tailer restarts from the end of the new file.
func (t *Tailer) SetFile(path string) {
	t.mu.Lock()
	changed := t.filePath != path
	t.filePath = path
	t.mu.Unlock()

	if changed && path != "" {
		log.Printf("tailer: now following %s", path)
	}
}

// CurrentFile returns the path being tailed.
func (t *Tailer) CurrentFile() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.filePath
}

// Run starts the tailing loop. It polls the current file for new lines.
// Call Stop() to terminate.
func (t *Tailer) Run() {
	var (
		currentPath string
		file        *os.File
		reader      *bufio.Reader
		offset      int64
	)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			if file != nil {
				file.Close()
			}
			return
		case <-ticker.C:
			t.mu.RLock()
			targetPath := t.filePath
			t.mu.RUnlock()

			if targetPath == "" {
				continue
			}

			// Handle file change
			if targetPath != currentPath {
				if file != nil {
					file.Close()
				}
				currentPath = targetPath
				f, err := os.Open(currentPath)
				if err != nil {
					log.Printf("tailer: error opening %s: %v", currentPath, err)
					file = nil
					continue
				}
				file = f
				// Seek to end — we only want new lines
				offset, _ = file.Seek(0, io.SeekEnd)
				reader = bufio.NewReader(file)
				continue
			}

			if file == nil {
				continue
			}

			// Check if file was truncated or rotated
			info, err := file.Stat()
			if err != nil {
				log.Printf("tailer: stat error: %v", err)
				file.Close()
				file = nil
				currentPath = ""
				continue
			}

			if info.Size() < offset {
				// File was truncated, re-read from start
				file.Seek(0, io.SeekStart)
				offset = 0
				reader = bufio.NewReader(file)
			}

			// Read new lines
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				offset += int64(len(line))
				t.mu.Lock()
				t.state.ParseLine(line)
				t.mu.Unlock()
			}
		}
	}
}

func (t *Tailer) Stop() {
	close(t.stopCh)
}
