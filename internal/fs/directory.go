package fs

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Directory represents a directory in the filesystem
type Directory struct {
	Path        string       // Full path to the directory
	Name        string       // Directory name without path
	IsReference bool         // Whether this is a reference directory
	ExcludePattern *regexp.Regexp // Pattern to exclude files
}

// NewDirectory creates a new Directory instance from a directory path
func NewDirectory(path string) (*Directory, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	dir := &Directory{
		Path: path,
		Name: filepath.Base(path),
	}

	return dir, nil
}

// SetExcludePattern sets the pattern to exclude files
func (d *Directory) SetExcludePattern(pattern string) error {
	if pattern == "" {
		d.ExcludePattern = nil
		return nil
	}

	// Convert glob patterns to regex
	regexStr := strings.Replace(pattern, ".", "\\.", -1)
	regexStr = strings.Replace(regexStr, "*", ".*", -1)
	regexStr = strings.Replace(regexStr, "?", ".", -1)
	regexStr = "^(" + strings.Replace(regexStr, ",", "|", -1) + ")$"
	
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return err
	}
	
	d.ExcludePattern = regex
	return nil
}

// ScanFiles scans the directory for files and returns them
func (d *Directory) ScanFiles(recursive bool) ([]*File, error) {
	var files []*File

	err := filepath.WalkDir(d.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories unless we're at the root
		if entry.IsDir() {
			if path != d.Path && !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches exclude pattern
		if d.ExcludePattern != nil && d.ExcludePattern.MatchString(entry.Name()) {
			return nil
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			return nil
		}

		// Create file object
		file := NewFileFromFileInfo(path, info)
		file.IsReference = d.IsReference

		// Add file to collection
		files = append(files, file)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// GetSubdirectories returns a list of subdirectories
func (d *Directory) GetSubdirectories() ([]*Directory, error) {
	var dirs []*Directory

	entries, err := os.ReadDir(d.Path)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subPath := filepath.Join(d.Path, entry.Name())
		subDir, err := NewDirectory(subPath)
		if err != nil {
			continue
		}

		subDir.IsReference = d.IsReference
		subDir.ExcludePattern = d.ExcludePattern
		dirs = append(dirs, subDir)
	}

	return dirs, nil
}
