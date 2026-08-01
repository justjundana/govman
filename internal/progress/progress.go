package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	_util "github.com/justjundana/govman/internal/util"
)

const (
	updateThreshold = 1 * time.Second // Print a line every 1 second max
)

type ProgressBar struct {
	total       int64
	current     int64
	description string
	startTime   time.Time
	lastUpdate  time.Time
	mutex       sync.Mutex
	finished    bool
	lastPct     int // last printed percentage (avoid duplicate lines)
	writer      io.Writer
	enabled     bool
	sessionBase int64
}

func New(total int64, description string) *ProgressBar {
	return NewWithWriter(total, description, os.Stderr, true)
}

// NewWithWriter creates a progress bar with an explicit destination and enabled state.
func NewWithWriter(total int64, description string, writer io.Writer, enabled bool) *ProgressBar {
	if writer == nil {
		writer = io.Discard
	}
	return &ProgressBar{
		total:       total,
		current:     0,
		description: description,
		startTime:   time.Now(),
		lastUpdate:  time.Time{}, // zero value so first render always fires
		lastPct:     -1,
		writer:      writer,
		enabled:     enabled,
	}
}

// IsTerminalWriter reports whether writer is an interactive character device.
func IsTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func (pb *ProgressBar) Write(p []byte) (n int, err error) {
	n = len(p)
	pb.Add(int64(n))
	return
}

func (pb *ProgressBar) Add(n int64) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	pb.current += n
	if pb.current < 0 {
		pb.current = 0
	}
	if pb.current > pb.total {
		pb.current = pb.total
	}

	now := time.Now()
	if now.Sub(pb.lastUpdate) >= updateThreshold || pb.current == pb.total {
		pb.render()
		pb.lastUpdate = now
	}
}

func (pb *ProgressBar) Set(current int64) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	pb.current = current
	if pb.current < 0 {
		pb.current = 0
	}
	if pb.current > pb.total {
		pb.current = pb.total
	}
	pb.sessionBase = pb.current
	pb.render()
}

func (pb *ProgressBar) Finish() {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	if pb.finished {
		return
	}

	pb.current = pb.total
	pb.finished = true
	pb.render()
}

func (pb *ProgressBar) render() {
	if !pb.enabled || pb.total <= 0 {
		return
	}

	pct := int(float64(pb.current) / float64(pb.total) * 100)
	if pct > 100 {
		pct = 100
	}

	// Skip if same percentage was already printed
	if pct == pb.lastPct {
		return
	}
	pb.lastPct = pct

	currentStr := _util.FormatBytes(pb.current)
	totalStr := _util.FormatBytes(pb.total)

	elapsed := time.Since(pb.startTime)
	var extra string
	if elapsed.Seconds() > 1 {
		transferred := pb.current - pb.sessionBase
		if transferred < 0 {
			transferred = 0
		}
		speed := float64(transferred) / elapsed.Seconds()
		speedStr := _util.FormatBytes(int64(speed)) + "/s"

		if speed > 0 && pb.current < pb.total {
			remaining := pb.total - pb.current
			eta := time.Duration(float64(remaining)/speed) * time.Second
			extra = fmt.Sprintf("  %s  ETA: %s", speedStr, _util.FormatDuration(eta))
		} else {
			extra = fmt.Sprintf("  %s", speedStr)
		}
	}

	_, _ = fmt.Fprintf(pb.writer, "  %3d%%  %s/%s%s\n", pct, currentStr, totalStr, extra)
}
