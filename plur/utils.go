package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// pluralize returns the singular or plural form of a word based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func toStdErr(dryRun bool, format string, args ...any) {
	if dryRun {
		format = "[dry-run] " + format
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

func dump(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Print(string(b))
}
