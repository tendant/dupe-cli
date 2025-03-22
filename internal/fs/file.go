package fs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tendant/dupe-cli/internal/hash"
)

// Constants for file operations
const (
	minPartialSize = hash.MinPartialSize
)

// File represents a file in the filesystem with metadata used for duplicate detection
type File struct {
	Path        string    // Full path to the file
	Name        string    // Filename without path
	Size        int64     // File size in bytes
	ModTime     time.Time // Last modification time
	Digest      []byte    // Full file hash (calculated on demand)
	DigestPart  []byte    // Partial file hash for large files (calculated on demand)
	Words       []string  // Words extracted from filename for fuzzy matching
	IsReference bool      // Whether this file is in a reference directory (shouldn't be deleted)
}

// NewFile creates a new File instance from a file path
func NewFile(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, os.ErrInvalid
	}

	file := &File{
		Path:    path,
		Name:    filepath.Base(path),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	return file, nil
}

// NewFileFromFileInfo creates a new File instance from os.FileInfo
func NewFileFromFileInfo(path string, info os.FileInfo) *File {
	return &File{
		Path:    path,
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}
}

// GetDigest returns the file's digest, calculating it if necessary
func (f *File) GetDigest() ([]byte, error) {
	if f.Digest != nil {
		return f.Digest, nil
	}

	digest, err := calculateFileHash(f.Path)
	if err != nil {
		return nil, err
	}

	f.Digest = digest
	return digest, nil
}

// GetPartialDigest returns a partial digest for large files, calculating it if necessary
func (f *File) GetPartialDigest() ([]byte, error) {
	if f.DigestPart != nil {
		return f.DigestPart, nil
	}

	// Only use partial digest for files larger than minPartialSize
	if f.Size < minPartialSize {
		return f.GetDigest()
	}

	digest, err := calculatePartialFileHash(f.Path)
	if err != nil {
		return nil, err
	}

	f.DigestPart = digest
	return digest, nil
}

// ExtractWords extracts words from the filename for fuzzy matching
func (f *File) ExtractWords() []string {
	if f.Words != nil {
		return f.Words
	}

	f.Words = extractWords(f.Name)
	return f.Words
}

// calculateFileHash calculates the hash of an entire file
func calculateFileHash(path string) ([]byte, error) {
	return hash.HashFile(path)
}

// calculatePartialFileHash calculates a partial hash of a file
func calculatePartialFileHash(path string) ([]byte, error) {
	return hash.HashFilePartial(path)
}

// extractWords extracts words from a filename for fuzzy matching
func extractWords(filename string) []string {
	// Convert to lowercase
	filename = strings.ToLower(filename)

	// Remove extension
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

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

	// Filter out very short words
	var words []string
	for _, part := range parts {
		if len(part) >= 2 {
			words = append(words, part)
		}
	}

	// If no words were extracted, use the whole filename
	if len(words) == 0 {
		words = append(words, strings.ToLower(filename))
	}

	return words
}
