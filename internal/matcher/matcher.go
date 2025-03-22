package matcher

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/tendant/dupe-cli/internal/fs"
)

// MatchType represents the type of matching to perform
type MatchType int

const (
	// MatchTypeExact is for exact matching
	MatchTypeExact MatchType = iota
	// MatchTypeFuzzy is for fuzzy matching
	MatchTypeFuzzy
)

// MatchOptions contains options for matching
type MatchOptions struct {
	Type            MatchType // Type of matching to perform
	MinMatchPercent int       // Minimum match percentage for fuzzy matching
	WeightByLength  bool      // Whether to weight words by length
	MatchSimilar    bool      // Whether to match similar words
}

// Match represents a match between two files
type Match struct {
	First      *fs.File // First file in the match
	Second     *fs.File // Second file in the match
	Percentage int      // Match percentage (0-100)
}

// Matcher is responsible for matching files
type Matcher struct {
	Options MatchOptions
}

// NewMatcher creates a new Matcher instance
func NewMatcher(options MatchOptions) *Matcher {
	return &Matcher{
		Options: options,
	}
}

// Match matches two files and returns the match percentage
func (m *Matcher) Match(first, second *fs.File) *Match {
	if m.Options.Type == MatchTypeExact {
		return m.matchExact(first, second)
	}
	return m.matchFuzzy(first, second)
}

// matchExact performs exact matching based on file content
func (m *Matcher) matchExact(first, second *fs.File) *Match {
	// If sizes are different, they can't be exact duplicates
	if first.Size != second.Size {
		return &Match{First: first, Second: second, Percentage: 0}
	}

	// For large files, first try partial hash
	if first.Size >= 3*1024*1024 { // 3MB
		digest1, err := first.GetPartialDigest()
		if err != nil {
			return &Match{First: first, Second: second, Percentage: 0}
		}

		digest2, err := second.GetPartialDigest()
		if err != nil {
			return &Match{First: first, Second: second, Percentage: 0}
		}

		// If partial hashes don't match, they're not duplicates
		if string(digest1) != string(digest2) {
			return &Match{First: first, Second: second, Percentage: 0}
		}
	}

	// Get full digests for final comparison
	digest1, err := first.GetDigest()
	if err != nil {
		return &Match{First: first, Second: second, Percentage: 0}
	}

	digest2, err := second.GetDigest()
	if err != nil {
		return &Match{First: first, Second: second, Percentage: 0}
	}

	// Compare digests
	if string(digest1) == string(digest2) {
		return &Match{First: first, Second: second, Percentage: 100}
	}

	return &Match{First: first, Second: second, Percentage: 0}
}

// matchFuzzy performs fuzzy matching based on filenames
func (m *Matcher) matchFuzzy(first, second *fs.File) *Match {
	// Extract words if not already done
	words1 := first.ExtractWords()
	words2 := second.ExtractWords()

	// Calculate match percentage
	percentage := m.compareWords(words1, words2)

	return &Match{First: first, Second: second, Percentage: percentage}
}

// compareWords compares two sets of words and returns the match percentage
func (m *Matcher) compareWords(first, second []string) int {
	if len(first) == 0 || len(second) == 0 {
		return 0
	}

	// Make a copy of second since we'll be removing items from it
	secondCopy := make([]string, len(second))
	copy(secondCopy, second)

	matchCount := 0
	totalCount := len(first) + len(second)

	// If weighting by length, adjust the total count
	if m.Options.WeightByLength {
		totalCount = 0
		for _, word := range first {
			totalCount += len(word)
		}
		for _, word := range second {
			totalCount += len(word)
		}
	}

	for _, word := range first {
		found := false

		// Try to find the word in the second list
		for i, secondWord := range secondCopy {
			if word == secondWord {
				// Remove the word from the second list to avoid matching it again
				secondCopy = append(secondCopy[:i], secondCopy[i+1:]...)
				found = true
				break
			}

			// If matching similar words is enabled, try to find similar words
			if m.Options.MatchSimilar && !found && isSimilar(word, secondWord) {
				secondCopy = append(secondCopy[:i], secondCopy[i+1:]...)
				found = true
				break
			}
		}

		if found {
			if m.Options.WeightByLength {
				matchCount += len(word)
			} else {
				matchCount++
			}
		}
	}

	// Calculate percentage
	if totalCount > 0 {
		percentage := (matchCount * 2 * 100) / totalCount
		if percentage > 100 {
			percentage = 100
		}
		return percentage
	}

	return 0
}

// isSimilar checks if two words are similar (used for fuzzy matching)
// This is a more aggressive implementation that considers words similar
// if they share a significant portion of characters
func isSimilar(word1, word2 string) bool {
	// If one is a substring of the other, they're similar
	if strings.Contains(word1, word2) || strings.Contains(word2, word1) {
		return true
	}

	// If they're both numbers, they're similar if they're close
	if isNumeric(word1) && isNumeric(word2) {
		// Consider numeric strings similar if they have the same length
		// This helps match things like "2023" and "2022"
		if len(word1) == len(word2) {
			return true
		}
	}

	// Count common characters
	commonChars := 0
	for _, c := range word1 {
		if strings.ContainsRune(word2, c) {
			commonChars++
		}
	}

	// Calculate similarity as percentage of common characters
	similarity := (commonChars * 100) / max(len(word1), len(word2))

	// Consider similar if at least 70% of characters are common
	return similarity >= 70
}

// isNumeric checks if a string contains only numeric characters
func isNumeric(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(s) > 0
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ExtractWords extracts words from a filename for fuzzy matching
func ExtractWords(filename string) []string {
	// Convert to lowercase
	filename = strings.ToLower(filename)

	// Remove extension
	filename = strings.TrimSuffix(filename, strings.ToLower(filepath.Ext(filename)))

	// Replace common separators with spaces
	replacers := []struct {
		old string
		new string
	}{
		{"-", " "},
		{"_", " "},
		{".", " "},
		{",", " "},
		{"(", " "},
		{")", " "},
		{"[", " "},
		{"]", " "},
		{"{", " "},
		{"}", " "},
	}

	for _, r := range replacers {
		filename = strings.ReplaceAll(filename, r.old, r.new)
	}

	// Split by whitespace
	parts := strings.Fields(filename)

	// Filter out non-alphanumeric words and very short words
	var words []string
	for _, part := range parts {
		if len(part) >= 2 && containsAlphaNumeric(part) {
			words = append(words, part)
		}
	}

	// If no words were extracted, use the whole filename
	if len(words) == 0 && len(filename) > 0 {
		words = append(words, filename)
	}

	return words
}

// containsAlphaNumeric checks if a string contains at least one alphanumeric character
func containsAlphaNumeric(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
