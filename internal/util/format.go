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

	value := float64(size)
	unitIndex := 0

	for i := range byteSizeUnits {
		value = value / unit
		unitIndex = i
		if value < unit || i == len(byteSizeUnits)-1 {
			break
		}
	}

	return fmt.Sprintf("%.0f %s", math.Round(value), byteSizeUnits[unitIndex])
}

// FormatDuration formats a time.Duration into a concise string (e.g., 45s, 3m12s, 2h05m).
// Parameter d is the duration. Returns a formatted string.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "-" + FormatDuration(-d)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
