package util

import (
	"fmt"
	"math"
	"time"
)

// Pre-allocated slice to avoid repeated allocations
var byteSizeUnits = []string{"KB", "MB", "GB", "TB", "PB", "EB"}

// FormatBytes converts a byte count into a human-readable string (KB, MB, GB, ...).
// Parameter size is the number of bytes. Returns a formatted string.
func FormatBytes(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size) / unit
	unitIndex := 0
	for value >= unit && unitIndex < len(byteSizeUnits)-1 {
		value /= unit
		unitIndex++
	}

	rounded := math.Round(value)
	if rounded >= unit && unitIndex < len(byteSizeUnits)-1 {
		rounded = math.Round(rounded / unit)
		unitIndex++
	}

	return fmt.Sprintf("%.0f %s", rounded, byteSizeUnits[unitIndex])
}

// FormatDuration formats a time.Duration into a concise string (e.g., 45s, 3m12s, 2h05m).
// Parameter d is the duration. Returns a formatted string.
func FormatDuration(d time.Duration) string {
	negative := d < 0
	var magnitude uint64
	if negative {
		// -(MinInt64) overflows time.Duration. Moving one step toward zero
		// before negating keeps the full magnitude representable as uint64.
		magnitude = uint64(-(d + 1)) + 1 // #nosec G115 -- the negative branch makes -(d+1) non-negative and bounded by MaxInt64.
	} else {
		magnitude = uint64(d)
	}

	secondsTotal := magnitude / uint64(time.Second)
	prefix := ""
	if negative {
		prefix = "-"
	}

	if magnitude < uint64(time.Minute) {
		return fmt.Sprintf("%s%ds", prefix, secondsTotal)
	}

	minutesTotal := secondsTotal / 60
	if magnitude < uint64(time.Hour) {
		return fmt.Sprintf("%s%dm%ds", prefix, minutesTotal, secondsTotal%60)
	}

	return fmt.Sprintf("%s%dh%dm", prefix, minutesTotal/60, minutesTotal%60)
}
