# Dupe CLI

A command-line tool written in Go that similar to the core functionality of [dupeGuru](https://github.com/arsenetar/dupeguru), a duplicate file finder.

## Features

- **Content-based matching**: Find exact duplicates based on file content
- **Fuzzy name matching**: Find similar files based on filename similarity
- **Recursive scanning**: Scan directories recursively
- **Exclusion patterns**: Skip files matching specific patterns
- **Multiple output formats**: Text, JSON, and CSV output formats
- **Space savings calculation**: See how much space you could save by removing duplicates
- **Optimized for large files**: Uses partial hashing for large files to improve performance

## Installation

### From Source

1. Ensure you have Go 1.18 or later installed
2. Clone this repository
3. Build the binary:

```bash
cd dupe-cli
go build -o dupe-cli ./cmd/dupe-cli
```

4. (Optional) Move the binary to a directory in your PATH:

```bash
sudo mv dupe-cli /usr/local/bin/
```

## Usage

```
Dupe CLI - Duplicate File Finder

Usage:
  dupe-cli [command] [flags]

Commands:
  scan        Scan directories for duplicate files
  help        Help about any command

Flags:
  -d, --directories string   Directories to scan (comma-separated)
  -r, --recursive            Scan directories recursively
  -m, --min-match int        Minimum match percentage for fuzzy matching (default: 80)
  -s, --scan-type string     Scan type (standard, content) (default: "standard")
  -e, --exclude string       Exclude patterns (comma-separated)
  -o, --output string        Output format (text, json, csv) (default: "text")
  -h, --help                 Help for dupe-cli
  -v, --version              Version for dupe-cli
```

## Examples

### Scan two directories using standard mode (fuzzy matching)

```bash
dupe-cli scan -d /path/to/dir1,/path/to/dir2
```

### Scan recursively with content-based matching

```bash
dupe-cli scan -d /path/to/dir -r -s content
```

### Exclude certain file patterns

```bash
dupe-cli scan -d /path/to/dir -e "*.tmp,*.log"
```

### Output results in JSON format

```bash
dupe-cli scan -d /path/to/dir -o json
```

### Set minimum match percentage for fuzzy matching

```bash
dupe-cli scan -d /path/to/dir -m 90
```

## Scan Types

- **standard**: Uses fuzzy matching based on filenames. Good for finding files with similar names that might be duplicates.
- **content**: Uses exact matching based on file content. Good for finding exact duplicates regardless of filename.

## Output Formats

- **text**: Human-readable text output
- **json**: JSON output for programmatic processing
- **csv**: CSV output for importing into spreadsheets

## How It Works

1. **File Scanning**: The tool scans the specified directories and collects file information.
2. **Grouping**: Files are grouped by size (files of different sizes cannot be duplicates).
3. **Matching**:
   - In content mode, files are compared using their hash values (MD5).
   - In standard mode, files are compared using fuzzy matching of their filenames.
4. **Results**: Duplicate groups are formed and displayed according to the specified output format.

## Optimization Techniques

- **Partial Hashing**: For large files, only portions of the file are hashed initially to quickly filter potential duplicates.
- **Size Grouping**: Files are first grouped by size to avoid unnecessary comparisons.
- **Word Extraction**: Filenames are broken down into words for more accurate fuzzy matching.

## License

MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is inspired by [dupeGuru](https://github.com/arsenetar/dupeguru) by Pascal Potvin and Andrew Senetar.
