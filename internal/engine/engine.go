package engine

import (
	"fmt"
	"sort"
	"sync"

	"github.com/tendant/dupe-cli/internal/fs"
	"github.com/tendant/dupe-cli/internal/matcher"
	"github.com/tendant/dupe-cli/internal/scanner"
)

// DuplicateGroup represents a group of duplicate files
type DuplicateGroup struct {
	Reference  *fs.File         // Reference file (original)
	Duplicates []*fs.File       // Duplicate files
	Matches    []*matcher.Match // Matches between reference and duplicates
}

// Engine is responsible for finding duplicates
type Engine struct {
	Scanner *scanner.Scanner
	Matcher *matcher.Matcher
	groups  []*DuplicateGroup
	mu      sync.Mutex
}

// NewEngine creates a new Engine instance
func NewEngine(scanner *scanner.Scanner, matcher *matcher.Matcher) *Engine {
	return &Engine{
		Scanner: scanner,
		Matcher: matcher,
		groups:  make([]*DuplicateGroup, 0),
	}
}

// FindDuplicates finds duplicate files
func (e *Engine) FindDuplicates() ([]*DuplicateGroup, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Scan directories
	_, err := e.Scanner.Scan()
	if err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	// Get potential duplicates (files with same size)
	potentialDupes := e.Scanner.GetPotentialDuplicates()

	// Process each group of potential duplicates
	e.groups = make([]*DuplicateGroup, 0)

	// Use a more sophisticated approach for grouping duplicates
	for _, files := range potentialDupes {
		e.processFileGroup(files)
	}

	// Sort groups by number of duplicates (descending)
	sort.Slice(e.groups, func(i, j int) bool {
		return len(e.groups[i].Duplicates) > len(e.groups[j].Duplicates)
	})

	return e.groups, nil
}

// processFileGroup processes a group of files with the same size
func (e *Engine) processFileGroup(files []*fs.File) {
	// Skip if less than 2 files
	if len(files) < 2 {
		return
	}

	// For exact matching, we can optimize by first grouping by hash
	if e.Matcher.Options.Type == matcher.MatchTypeExact {
		e.processExactMatches(files)
	} else {
		e.processFuzzyMatches(files)
	}
}

// processExactMatches processes files using exact matching (hash-based)
func (e *Engine) processExactMatches(files []*fs.File) {
	// Group files by hash
	filesByHash := make(map[string][]*fs.File)

	for _, file := range files {
		// Get hash (partial for large files, full for small files)
		var hash []byte
		var err error

		if file.Size >= 3*1024*1024 { // 3MB
			hash, err = file.GetPartialDigest()
		} else {
			hash, err = file.GetDigest()
		}

		if err != nil {
			continue
		}

		hashStr := string(hash)
		filesByHash[hashStr] = append(filesByHash[hashStr], file)
	}

	// Process each hash group
	for _, hashGroup := range filesByHash {
		if len(hashGroup) < 2 {
			continue
		}

		// For files with the same partial hash, verify with full hash
		if hashGroup[0].Size >= 3*1024*1024 {
			filesByFullHash := make(map[string][]*fs.File)

			for _, file := range hashGroup {
				hash, err := file.GetDigest()
				if err != nil {
					continue
				}

				hashStr := string(hash)
				filesByFullHash[hashStr] = append(filesByFullHash[hashStr], file)
			}

			// Create groups for each full hash match
			for _, fullHashGroup := range filesByFullHash {
				if len(fullHashGroup) < 2 {
					continue
				}

				e.createDuplicateGroup(fullHashGroup)
			}
		} else {
			// For small files, we already have the full hash
			e.createDuplicateGroup(hashGroup)
		}
	}
}

// processFuzzyMatches processes files using fuzzy matching (filename-based)
func (e *Engine) processFuzzyMatches(files []*fs.File) {
	// Use a more sophisticated approach for fuzzy matching
	// We'll compare each file with every other file and build clusters of matches

	// Track which files have been processed
	processed := make(map[*fs.File]bool)

	for _, file := range files {
		if processed[file] {
			continue
		}

		// Find all files that match with this file
		matches := make([]*matcher.Match, 0)
		duplicates := make([]*fs.File, 0)

		for _, otherFile := range files {
			if file == otherFile || processed[otherFile] {
				continue
			}

			match := e.Matcher.Match(file, otherFile)
			if match.Percentage >= e.Matcher.Options.MinMatchPercent {
				matches = append(matches, match)
				duplicates = append(duplicates, otherFile)
				processed[otherFile] = true
			}
		}

		// If duplicates found, create a group
		if len(duplicates) > 0 {
			processed[file] = true

			group := &DuplicateGroup{
				Reference:  file,
				Duplicates: duplicates,
				Matches:    matches,
			}

			e.groups = append(e.groups, group)
		}
	}
}

// createDuplicateGroup creates a duplicate group from a list of files
func (e *Engine) createDuplicateGroup(files []*fs.File) {
	// Use the first file as reference
	reference := files[0]
	duplicates := files[1:]

	// Create matches
	matches := make([]*matcher.Match, 0, len(duplicates))
	for _, dupe := range duplicates {
		match := e.Matcher.Match(reference, dupe)
		matches = append(matches, match)
	}

	// Create group
	group := &DuplicateGroup{
		Reference:  reference,
		Duplicates: duplicates,
		Matches:    matches,
	}

	e.groups = append(e.groups, group)
}

// GetGroups returns the duplicate groups
func (e *Engine) GetGroups() []*DuplicateGroup {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.groups
}

// GetTotalDuplicateCount returns the total number of duplicate files
func (e *Engine) GetTotalDuplicateCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	count := 0
	for _, group := range e.groups {
		count += len(group.Duplicates)
	}
	return count
}

// GetTotalDuplicateSize returns the total size of duplicate files
func (e *Engine) GetTotalDuplicateSize() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	var size int64
	for _, group := range e.groups {
		for _, dupe := range group.Duplicates {
			size += dupe.Size
		}
	}
	return size
}

// FilterGroups filters duplicate groups based on a predicate
func (e *Engine) FilterGroups(predicate func(*DuplicateGroup) bool) []*DuplicateGroup {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]*DuplicateGroup, 0)
	for _, group := range e.groups {
		if predicate(group) {
			result = append(result, group)
		}
	}
	return result
}
