package main

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// rspecExampleRegex matches the start of an rspec example declaration:
// `it`, `specify`, `example`, `scenario`, `focus`, `fit`, `fexample`,
// `fspecify`, `pending`, `xit`, `xspecify`, `xexample`. Matches when the
// keyword starts the (optionally indented) line and is followed by a space,
// open-paren, or open-brace.
var rspecExampleRegex = regexp.MustCompile(`^\s*(?:it|specify|example|scenario|focus|fit|fexample|fspecify|pending|xit|xspecify|xexample)[\s\(\{]`)

// findExampleLines returns the 1-based line numbers of rspec example
// declarations in filePath. Falls back to nil on read errors.
func findExampleLines(filePath string) ([]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []int
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if rspecExampleRegex.MatchString(scanner.Text()) {
			lines = append(lines, lineNum)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// splitFileByExamples splits a spec file into numChunks rspec invocations by
// distributing example line numbers round-robin across chunks. Each returned
// element is a string of form "path:line1:line2:..." that rspec accepts as a
// positional file argument.
//
// Returns (paths, true) on success, ([originalPath], false) when the file
// can't usefully be split (too few examples or unreadable).
func splitFileByExamples(filePath string, numChunks int) ([]string, bool) {
	if numChunks <= 1 {
		return []string{filePath}, false
	}
	lines, err := findExampleLines(filePath)
	if err != nil || len(lines) < numChunks*4 {
		return []string{filePath}, false
	}

	builders := make([]strings.Builder, numChunks)
	for i, line := range lines {
		idx := i % numChunks
		if builders[idx].Len() == 0 {
			builders[idx].WriteString(filePath)
		}
		builders[idx].WriteByte(':')
		builders[idx].WriteString(strconv.Itoa(line))
	}

	result := make([]string, 0, numChunks)
	for i := range builders {
		s := builders[i].String()
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) < 2 {
		return []string{filePath}, false
	}
	return result, true
}
