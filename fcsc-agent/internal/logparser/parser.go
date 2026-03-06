package logparser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// QueueStats holds parsed values from a single QUEUE log line.
type QueueStats struct {
	Name     string
	TotalIn  int64
	TotalOut int64
	Backlog  int64
	SendRate float64
	RecvRate float64
}

// ParsedState holds the latest parsed state from all log lines.
type ParsedState struct {
	Queues           map[string]*QueueStats
	LastGeyserUpdate time.Time
	LastLineTime     time.Time
}

func NewParsedState() *ParsedState {
	return &ParsedState{
		Queues: make(map[string]*QueueStats),
	}
}

var (
	// Matches: QUEUE: geyser subscribe - Total In: 122 - Total Out: 122 - Backlog: 0 - Send Rate: 0 msg/s - Recv Rate: 0 msg/s
	queuePattern = regexp.MustCompile(
		`QUEUE:\s+(.+?)\s+-\s+Total In:\s+(\d+)\s+-\s+Total Out:\s+(\d+)\s+-\s+Backlog:\s+(\d+)\s+-\s+Send Rate:\s+(\d+)\s+msg/s\s+-\s+Recv Rate:\s+(\d+)\s+msg/s`,
	)

	// Matches: in_geyser timestamp: 2026-03-04 07:50:53.577816795 UTC
	geyserTimestampPattern = regexp.MustCompile(
		`in_geyser timestamp:\s+(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\.\d+)\s+UTC`,
	)

	// Matches the leading timestamp: 2026-03-04T07:50:32.338225Z
	lineTimestampPattern = regexp.MustCompile(
		`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)`,
	)
)

// ParseLine processes a single log line and updates the state.
// Returns true if the line was meaningful (matched a pattern).
func (s *ParsedState) ParseLine(line string) bool {
	matched := false

	// Parse leading timestamp
	if m := lineTimestampPattern.FindStringSubmatch(line); m != nil {
		if t, err := time.Parse("2006-01-02T15:04:05.999999999Z", m[1]); err == nil {
			s.LastLineTime = t
		}
	}

	// Parse QUEUE lines
	if m := queuePattern.FindStringSubmatch(line); m != nil {
		name := normalizeQueueName(m[1])
		totalIn, _ := strconv.ParseInt(m[2], 10, 64)
		totalOut, _ := strconv.ParseInt(m[3], 10, 64)
		backlog, _ := strconv.ParseInt(m[4], 10, 64)
		sendRate, _ := strconv.ParseFloat(m[5], 64)
		recvRate, _ := strconv.ParseFloat(m[6], 64)

		s.Queues[name] = &QueueStats{
			Name:     name,
			TotalIn:  totalIn,
			TotalOut: totalOut,
			Backlog:  backlog,
			SendRate: sendRate,
			RecvRate: recvRate,
		}
		matched = true
	}

	// Parse geyser timestamp
	if m := geyserTimestampPattern.FindStringSubmatch(line); m != nil {
		if t, err := time.Parse("2006-01-02 15:04:05.999999999", m[1]); err == nil {
			s.LastGeyserUpdate = t
		}
		matched = true
	}

	return matched
}

func normalizeQueueName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
