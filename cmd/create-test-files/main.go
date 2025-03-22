package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// createTestFiles creates test files for testing the duplicate detection
func createTestFiles() error {
	// Create test directory
	testDir := filepath.Join(os.TempDir(), "dupe-cli-test")

	// Clean up any existing test directory
	os.RemoveAll(testDir)

	// Create directories
	dirs := []string{
		filepath.Join(testDir, "dir1"),
		filepath.Join(testDir, "dir2"),
		filepath.Join(testDir, "dir3"),
	}

	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create exact duplicate files (same content)
	content1 := []byte("This is test file content for exact duplicates.\n")
	err := os.WriteFile(filepath.Join(dirs[0], "file1.txt"), content1, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dirs[1], "file1_copy.txt"), content1, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dirs[2], "another_copy.txt"), content1, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Create similar named files (for fuzzy matching)
	err = os.WriteFile(filepath.Join(dirs[0], "document-2023.txt"), []byte("Different content 1"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dirs[1], "document_2023.txt"), []byte("Different content 2"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Create larger duplicate files
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err = os.WriteFile(filepath.Join(dirs[0], "large_file.bin"), largeContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dirs[2], "large_file_copy.bin"), largeContent, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Create unique files
	err = os.WriteFile(filepath.Join(dirs[0], "unique1.txt"), []byte("Unique content 1"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	err = os.WriteFile(filepath.Join(dirs[1], "unique2.txt"), []byte("Unique content 2"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	fmt.Printf("Test files created in %s\n", testDir)
	fmt.Printf("To test the tool, run:\n")
	fmt.Printf("  dupe-cli scan -d %s -r\n", testDir)
	fmt.Printf("  dupe-cli scan -d %s -r -s content\n", testDir)

	return nil
}

func main() {
	err := createTestFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
