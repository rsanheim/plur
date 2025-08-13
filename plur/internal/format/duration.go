package format

import (
	"fmt"
	"math"
	"strings"
)

// FormatDuration formats a duration in seconds using RSpec's adaptive precision rules:
// - < 1 second: up to 5 decimal places (trailing zeros removed)
// - 1-119 seconds: up to 2 decimal places (trailing zeros removed)
// - 120-299 seconds: 1 decimal place
// - >= 300 seconds: no decimal places
// - > 60 seconds: formatted as "X minutes Y seconds"
func FormatDuration(seconds float64) string {
	if seconds < 0 {
		return "0 seconds"
	}

	// Determine precision based on duration
	var precision int
	switch {
	case seconds < 1:
		precision = 5
	case seconds < 120:
		precision = 2
	case seconds < 300:
		precision = 1
	default:
		precision = 0
	}

	// Format durations over 60 seconds as minutes + seconds
	if seconds >= 60 {
		// First round the seconds according to precision
		multiplier := math.Pow(10, float64(precision))
		roundedTotal := math.Round(seconds*multiplier) / multiplier

		minutes := int(roundedTotal / 60)
		remainingSeconds := roundedTotal - float64(minutes*60)

		minuteStr := pluralize(minutes, "minute")
		secondStr := pluralize(formatSeconds(remainingSeconds, precision), "second")

		return fmt.Sprintf("%s %s", minuteStr, secondStr)
	}

	// Format durations under 60 seconds
	return pluralize(formatSeconds(seconds, precision), "second")
}

// formatSeconds formats seconds with the specified precision, removing trailing zeros
func formatSeconds(seconds float64, precision int) string {
	// Round to the specified precision
	multiplier := math.Pow(10, float64(precision))
	rounded := math.Round(seconds*multiplier) / multiplier

	// Format with the specified precision
	formatted := fmt.Sprintf("%.*f", precision, rounded)

	// Strip trailing zeros (but keep at least one digit after decimal if there's a decimal point)
	return stripTrailingZeros(formatted)
}

// stripTrailingZeros removes trailing zeros from a formatted number
// Examples: "1.00" -> "1", "1.50" -> "1.5", "1.0001" -> "1.0001"
func stripTrailingZeros(s string) string {
	// If there's no decimal point, return as-is
	if !strings.Contains(s, ".") {
		return s
	}

	// Remove trailing zeros after decimal point
	s = strings.TrimRight(s, "0")

	// If we removed all digits after decimal, remove the decimal too
	s = strings.TrimRight(s, ".")

	return s
}

// pluralize returns a pluralized string with count
// Examples: pluralize(1, "second") -> "1 second", pluralize(2, "second") -> "2 seconds"
func pluralize(count interface{}, word string) string {
	var countStr string
	var isOne bool

	switch v := count.(type) {
	case int:
		countStr = fmt.Sprintf("%d", v)
		isOne = v == 1
	case string:
		countStr = v
		// Check if the string represents 1 (could be "1", "1.0", etc)
		isOne = v == "1" || v == "1.0"
	default:
		countStr = fmt.Sprintf("%v", v)
		isOne = false
	}

	if isOne {
		return fmt.Sprintf("%s %s", countStr, word)
	}

	// Handle words ending in 's' (like "process")
	if strings.HasSuffix(word, "s") {
		return fmt.Sprintf("%s %ses", countStr, word)
	}

	return fmt.Sprintf("%s %ss", countStr, word)
}
