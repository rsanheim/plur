package rspec

import "regexp"

// These suffixes follow RSpec's extract_location and Example.parse_id syntax.
var targetSelector = regexp.MustCompile(`(?::[0-9]+)+$|\[[0-9\s:,]+\]$`)

// TargetPath removes a line or scoped-example selector without accessing the file.
func TargetPath(target string) string {
	if loc := targetSelector.FindStringIndex(target); loc != nil {
		return target[:loc[0]]
	}
	return target
}
