package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/tendant/dupe-cli/internal/fs"
)

// ScanType represents the type of scan to perform
type ScanType int

const (
	// ScanTypeStandard is the standard scan type (filename-based)
	ScanTypeStandard ScanType = iota
	// ScanTypeContent is the content scan type (hash-based)
	ScanTypeContent
)

// Scanner is responsible for scanning directories and finding files
type Scanner struct {
	Directories    []string             // Directories to scan
	ExcludePattern *regexp.Regexp       // Pattern to exclude files
	Recursive      bool                 // Whether to scan recursively
	ScanType       ScanType             // Type of scan to perform
	MinMatchPct    int                  // Minimum match percentage for fuzzy matching
	RefDirs        map[string]bool      // Reference directories (files won't be marked for deletion)
	mu             sync.Mutex           // Mutex for thread safety
	files          []*fs.File           // Collected files
	filesBySize    map[int64][]*fs.File // Files grouped by size
}

// NewScanner creates a new Scanner instance
func NewScanner(dirs []string, exclude string, recursive bool, scanType ScanType, minMatch int) *Scanner {
	var excludePattern *regexp.Regexp
	if exclude != "" {
		// Convert glob patterns to regex
		regexStr := strings.Replace(exclude, ".", "\\.", -1)
		regexStr = strings.Replace(regexStr, "*", ".*", -1)
		regexStr = strings.Replace(regexStr, "?", ".", -1)
		regexStr = "^(" + strings.Replace(regexStr, ",", "|", -1) + ")$"
		excludePattern = regexp.MustCompile(regexStr)
	}

	return &Scanner{
		Directories:    dirs,
		ExcludePattern: excludePattern,
		Recursive:      recursive,
		ScanType:       scanType,
		MinMatchPct:    minMatch,
		RefDirs:        make(map[string]bool),
		filesBySize:    make(map[int64][]*fs.File),
	}
}

// SetReferenceDir marks a directory as a reference directory
func (s *Scanner) SetReferenceDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RefDirs[dir] = true
}

// Scan scans the directories and returns the files
func (s *Scanner) Scan() ([]*fs.File, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.files = make([]*fs.File, 0)
	s.filesBySize = make(map[int64][]*fs.File)

	for _, dirPath := range s.Directories {
		// Create directory object
		dir, err := fs.NewDirectory(dirPath)
		if err != nil {
			return nil, fmt.Errorf("error creating directory object for %s: %w", dirPath, err)
		}

		// Set exclude pattern
		if s.ExcludePattern != nil {
			dir.ExcludePattern = s.ExcludePattern
		}

		// Check if this is a reference directory
		if s.RefDirs[dirPath] {
			dir.IsReference = true
		}

		// Scan directory for files
		err = s.scanDirectory(dir)
		if err != nil {
			return nil, fmt.Errorf("error scanning directory %s: %w", dirPath, err)
		}
	}

	return s.files, nil
}

// scanDirectory scans a directory for files
func (s *Scanner) scanDirectory(dir *fs.Directory) error {
	// Scan files in this directory
	files, err := dir.ScanFiles(s.Recursive)
	if err != nil {
		return err
	}

	// Process files
	for _, file := range files {
		// Add file to collection
		s.files = append(s.files, file)

		// Group files by size (files of different sizes cannot be duplicates)
		s.filesBySize[file.Size] = append(s.filesBySize[file.Size], file)
	}

	return nil
}

// GetFilesBySize returns files grouped by size
func (s *Scanner) GetFilesBySize() map[int64][]*fs.File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.filesBySize
}

// GetPotentialDuplicates returns files that have the same size (potential duplicates)
func (s *Scanner) GetPotentialDuplicates() [][]*fs.File {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([][]*fs.File, 0)
	for _, files := range s.filesBySize {
		if len(files) > 1 {
			result = append(result, files)
		}
	}
	return result
}

// GetFiles returns all scanned files
func (s *Scanner) GetFiles() []*fs.File {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.files
}

// GetFileCount returns the number of scanned files
func (s *Scanner) GetFileCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.files)
}

// ScanFile scans a single file and adds it to the collection
func (s *Scanner) ScanFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Check if it's a directory
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}

	// Check if file matches exclude pattern
	if s.ExcludePattern != nil && s.ExcludePattern.MatchString(filepath.Base(path)) {
		return nil
	}

	// Create file object
	file := fs.NewFileFromFileInfo(path, info)

	// Check if file is in a reference directory
	for refDir := range s.RefDirs {
		if strings.HasPrefix(path, refDir) {
			file.IsReference = true
			break
		}
	}

	// Add file to collection
	s.files = append(s.files, file)

	// Group files by size
	s.filesBySize[info.Size()] = append(s.filesBySize[info.Size()], file)

	return nil
}
