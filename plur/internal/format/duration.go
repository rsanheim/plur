package format

import (
	"fmt"
	"math"
	"strconv"
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

	if seconds >= 60 {
		multiplier := math.Pow(10, float64(precision))
		roundedTotal := math.Round(seconds*multiplier) / multiplier

		minutes := int(roundedTotal / 60)
		remainingSeconds := roundedTotal - float64(minutes*60)

		minuteStr := pluralize(strconv.Itoa(minutes), "minute")
		secondStr := pluralize(formatSeconds(remainingSeconds, precision), "second")

		return fmt.Sprintf("%s %s", minuteStr, secondStr)
	}

	return pluralize(formatSeconds(seconds, precision), "second")
}

func formatSeconds(seconds float64, precision int) string {
	multiplier := math.Pow(10, float64(precision))
	rounded := math.Round(seconds*multiplier) / multiplier

	if precision == 0 {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}

	result := strconv.FormatFloat(rounded, 'g', -1, 64)

	// Prevent scientific notation for whole numbers
	if rounded == math.Trunc(rounded) && rounded >= 1 {
		result = strconv.FormatFloat(rounded, 'f', 0, 64)
	}

	return result
}

func pluralize(countStr string, word string) string {
	isOne := countStr == "1" || countStr == "1.0"

	if isOne {
		return fmt.Sprintf("%s %s", countStr, word)
	}

	// Handle words ending in 's' (like "process")
	if strings.HasSuffix(word, "s") {
		return fmt.Sprintf("%s %ses", countStr, word)
	}

	return fmt.Sprintf("%s %ss", countStr, word)
}
