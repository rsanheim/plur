package main

import (
	"fmt"
	"path"
	"strings"
)

const watchSharedHelperNoRuleHint = "[watch] Hint: add a [[watch]] mapping for shared files if this change should run tests."

func printWatchNoRule(filePath string) {
	fmt.Printf("[watch] No matching rule for %s\n", filePath)
	if shouldHintWatchSharedHelper(filePath) {
		fmt.Println(watchSharedHelperNoRuleHint)
	}
}

func shouldHintWatchSharedHelper(filePath string) bool {
	normalized := path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "./")

	if !strings.HasSuffix(normalized, ".rb") {
		return false
	}

	if strings.HasPrefix(normalized, "spec/") {
		return !strings.HasSuffix(normalized, "_spec.rb")
	}
	if strings.HasPrefix(normalized, "test/") {
		return !strings.HasSuffix(normalized, "_test.rb")
	}
	return false
}
