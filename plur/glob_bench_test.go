package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
)

// Test fixtures setup
func setupBenchmarkFixtures(b *testing.B) string {
	b.Helper()

	// Use the rspec reference project for benchmarking - much larger dataset
	// This has ~225 spec files vs ~12 in the fixture project
	fixtureDir := "../references/rspec"

	// Verify it exists
	if _, err := os.Stat(fixtureDir); os.IsNotExist(err) {
		b.Fatalf("Fixture directory not found: %s", fixtureDir)
	}

	// Change to fixture directory
	originalDir, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get current directory: %v", err)
	}

	err = os.Chdir(fixtureDir)
	if err != nil {
		b.Fatalf("Failed to change to fixture directory: %v", err)
	}

	return originalDir
}

func BenchmarkExpandGlobPatterns_Simple(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	// Simple pattern in one of the sub-projects (one level deep)
	patterns := []string{"rspec-core/spec/rspec/core/*_spec.rb"}
	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ExpandGlobPatterns(patterns, framework)
		if err != nil {
			b.Fatalf("ExpandGlobPatterns failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

func BenchmarkExpandGlobPatterns_Recursive(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	// Recursive pattern across entire rspec monorepo (~225 files)
	patterns := []string{"**/*_spec.rb"}
	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ExpandGlobPatterns(patterns, framework)
		if err != nil {
			b.Fatalf("ExpandGlobPatterns failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

func BenchmarkExpandGlobPatterns_Multiple(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	// Use actual rspec project subdirectories
	patterns := []string{"rspec-core/**/*_spec.rb", "rspec-mocks/**/*_spec.rb"}
	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ExpandGlobPatterns(patterns, framework)
		if err != nil {
			b.Fatalf("ExpandGlobPatterns failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

func BenchmarkExpandGlobPatterns_Directory(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	// Test directory expansion on large subdirectory
	patterns := []string{"rspec-core/spec/"}
	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ExpandGlobPatterns(patterns, framework)
		if err != nil {
			b.Fatalf("ExpandGlobPatterns failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

func BenchmarkExpandGlobPatterns_LargeRecursive(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	// Benchmark finding all specs in one large sub-project
	patterns := []string{"rspec-core/**"}
	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := ExpandGlobPatterns(patterns, framework)
		if err != nil {
			b.Fatalf("ExpandGlobPatterns failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

func BenchmarkFindTestFiles_RSpec(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	framework := FrameworkRSpec

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := FindTestFiles(framework)
		if err != nil {
			b.Fatalf("FindTestFiles failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

// Benchmark doublestar directly for comparison
func BenchmarkDoublestarGlob(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	pattern := "**/*_spec.rb"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			b.Fatalf("doublestar.FilepathGlob failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}

// Benchmark stdlib filepath.Glob for comparison
func BenchmarkFilepathGlob_Simple(b *testing.B) {
	originalDir := setupBenchmarkFixtures(b)
	defer os.Chdir(originalDir)

	pattern := "rspec-core/spec/rspec/core/*_spec.rb"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := filepath.Glob(pattern)
		if err != nil {
			b.Fatalf("filepath.Glob failed: %v", err)
		}
		if len(files) == 0 {
			b.Fatal("No files found")
		}
	}
}
