package main

// pluralize returns the singular or plural form of a word based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
