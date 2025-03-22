package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tendant/dupe-cli/internal/engine"
	"github.com/tendant/dupe-cli/internal/matcher"
	"github.com/tendant/dupe-cli/internal/scanner"
)

// Version information
const (
	Version = "0.1.0"
)

// Command line flags
type Flags struct {
	Directories    []string
	Recursive      bool
	ExcludePattern string
	ScanType       string
	MinMatchPct    int
	OutputFormat   string
	Help           bool
	Version        bool
}

// Result formats
type ResultFormat int

const (
	ResultFormatText ResultFormat = iota
	ResultFormatJSON
	ResultFormatCSV
)

func main() {
	// Parse command line arguments
	flags, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	// Show help
	if flags.Help {
		printUsage()
		os.Exit(0)
	}

	// Show version
	if flags.Version {
		fmt.Printf("dupe-cli version %s\n", Version)
		os.Exit(0)
	}

	// Validate arguments
	if len(flags.Directories) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No directories specified\n")
		printUsage()
		os.Exit(1)
	}

	// Validate directories
	for _, dir := range flags.Directories {
		info, err := os.Stat(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Directory not found: %s\n", dir)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: Not a directory: %s\n", dir)
			os.Exit(1)
		}
	}

	// Run scan
	err = runScan(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseArgs parses command line arguments
func parseArgs(args []string) (*Flags, error) {
	flags := &Flags{
		Directories:  []string{},
		Recursive:    false,
		ScanType:     "standard",
		MinMatchPct:  80,
		OutputFormat: "text",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "scan":
			// Command, just skip
			continue

		case arg == "-h" || arg == "--help":
			flags.Help = true

		case arg == "-v" || arg == "--version":
			flags.Version = true

		case arg == "-r" || arg == "--recursive":
			flags.Recursive = true

		case arg == "-d" || arg == "--directories":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			dirs := strings.Split(args[i], ",")
			for _, dir := range dirs {
				flags.Directories = append(flags.Directories, strings.TrimSpace(dir))
			}

		case arg == "-e" || arg == "--exclude":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags.ExcludePattern = args[i]

		case arg == "-s" || arg == "--scan-type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags.ScanType = strings.ToLower(args[i])
			if flags.ScanType != "standard" && flags.ScanType != "content" {
				return nil, fmt.Errorf("invalid scan type: %s", flags.ScanType)
			}

		case arg == "-m" || arg == "--min-match":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			pct, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid min match percentage: %s", args[i])
			}
			if pct < 0 || pct > 100 {
				return nil, fmt.Errorf("min match percentage must be between 0 and 100")
			}
			flags.MinMatchPct = pct

		case arg == "-o" || arg == "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			flags.OutputFormat = strings.ToLower(args[i])
			if flags.OutputFormat != "text" && flags.OutputFormat != "json" && flags.OutputFormat != "csv" {
				return nil, fmt.Errorf("invalid output format: %s", flags.OutputFormat)
			}

		case strings.HasPrefix(arg, "-"):
			return nil, fmt.Errorf("unknown option: %s", arg)

		default:
			// Assume it's a directory
			flags.Directories = append(flags.Directories, arg)
		}
	}

	return flags, nil
}

// printUsage prints usage information
func printUsage() {
	fmt.Println("Dupe CLI - Duplicate File Finder")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  dupe-cli [command] [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  scan        Scan directories for duplicate files")
	fmt.Println("  help        Help about any command")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -d, --directories string   Directories to scan (comma-separated)")
	fmt.Println("  -r, --recursive            Scan directories recursively")
	fmt.Println("  -m, --min-match int        Minimum match percentage for fuzzy matching (default: 80)")
	fmt.Println("  -s, --scan-type string     Scan type (standard, content) (default: \"standard\")")
	fmt.Println("  -e, --exclude string       Exclude patterns (comma-separated)")
	fmt.Println("  -o, --output string        Output format (text, json, csv) (default: \"text\")")
	fmt.Println("  -h, --help                 Help for dupe-cli")
	fmt.Println("  -v, --version              Version for dupe-cli")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Scan two directories using standard mode")
	fmt.Println("  dupe-cli scan -d /path/to/dir1,/path/to/dir2")
	fmt.Println("")
	fmt.Println("  # Scan recursively with content-based matching")
	fmt.Println("  dupe-cli scan -d /path/to/dir -r -s content")
	fmt.Println("")
	fmt.Println("  # Exclude certain file patterns")
	fmt.Println("  dupe-cli scan -d /path/to/dir -e \"*.tmp,*.log\"")
	fmt.Println("")
	fmt.Println("  # Output results in JSON format")
	fmt.Println("  dupe-cli scan -d /path/to/dir -o json")
}

// runScan runs the scan with the specified flags
func runScan(flags *Flags) error {
	startTime := time.Now()

	// Convert scan type string to ScanType
	var scanType scanner.ScanType
	switch flags.ScanType {
	case "content":
		scanType = scanner.ScanTypeContent
	default:
		scanType = scanner.ScanTypeStandard
	}

	// Create scanner
	s := scanner.NewScanner(flags.Directories, flags.ExcludePattern, flags.Recursive, scanType, flags.MinMatchPct)

	// Create matcher
	matchOpts := matcher.MatchOptions{
		MinMatchPercent: flags.MinMatchPct,
		WeightByLength:  true,
		MatchSimilar:    true,
	}

	if scanType == scanner.ScanTypeContent {
		matchOpts.Type = matcher.MatchTypeExact
	} else {
		matchOpts.Type = matcher.MatchTypeFuzzy
	}

	m := matcher.NewMatcher(matchOpts)

	// Create engine
	e := engine.NewEngine(s, m)

	// Print scan start message
	fmt.Printf("Scanning directories: %s\n", strings.Join(flags.Directories, ", "))
	fmt.Printf("Scan type: %s\n", flags.ScanType)
	if flags.Recursive {
		fmt.Println("Recursive: yes")
	} else {
		fmt.Println("Recursive: no")
	}
	if flags.ExcludePattern != "" {
		fmt.Printf("Exclude pattern: %s\n", flags.ExcludePattern)
	}
	fmt.Printf("Minimum match percentage: %d%%\n", flags.MinMatchPct)
	fmt.Println("Scanning...")

	// Find duplicates
	groups, err := e.FindDuplicates()
	if err != nil {
		return err
	}

	// Calculate scan time
	scanTime := time.Since(startTime)

	// Output results
	switch flags.OutputFormat {
	case "json":
		return outputJSON(groups, e.GetTotalDuplicateCount(), e.GetTotalDuplicateSize(), scanTime)
	case "csv":
		return outputCSV(groups, e.GetTotalDuplicateCount(), e.GetTotalDuplicateSize(), scanTime)
	default:
		return outputText(groups, e.GetTotalDuplicateCount(), e.GetTotalDuplicateSize(), scanTime)
	}
}

// outputText outputs results in text format
func outputText(groups []*engine.DuplicateGroup, totalDupes int, totalSize int64, scanTime time.Duration) error {
	fmt.Printf("\nScan completed in %s\n", scanTime)
	fmt.Printf("Found %d duplicate groups with %d total duplicates\n", len(groups), totalDupes)
	fmt.Printf("Total space that could be freed: %s\n", formatSize(totalSize))

	if len(groups) == 0 {
		fmt.Println("No duplicates found.")
		return nil
	}

	for i, group := range groups {
		fmt.Printf("\nGroup %d:\n", i+1)
		fmt.Printf("  Reference: %s (%s)\n", group.Reference.Path, formatSize(group.Reference.Size))

		for j, dupe := range group.Duplicates {
			match := group.Matches[j]
			fmt.Printf("  Duplicate %d: %s (%s, %d%% match)\n",
				j+1, dupe.Path, formatSize(dupe.Size), match.Percentage)
		}
	}

	return nil
}

// outputJSON outputs results in JSON format
func outputJSON(groups []*engine.DuplicateGroup, totalDupes int, totalSize int64, scanTime time.Duration) error {
	type Match struct {
		Path       string `json:"path"`
		Size       int64  `json:"size"`
		Percentage int    `json:"percentage"`
	}

	type Group struct {
		Reference  string  `json:"reference"`
		RefSize    int64   `json:"reference_size"`
		Duplicates []Match `json:"duplicates"`
	}

	type Result struct {
		ScanTime       string  `json:"scan_time"`
		GroupCount     int     `json:"group_count"`
		DuplicateCount int     `json:"duplicate_count"`
		TotalSize      int64   `json:"total_size"`
		Groups         []Group `json:"groups"`
	}

	result := Result{
		ScanTime:       scanTime.String(),
		GroupCount:     len(groups),
		DuplicateCount: totalDupes,
		TotalSize:      totalSize,
		Groups:         make([]Group, 0, len(groups)),
	}

	for _, group := range groups {
		g := Group{
			Reference:  group.Reference.Path,
			RefSize:    group.Reference.Size,
			Duplicates: make([]Match, 0, len(group.Duplicates)),
		}

		for j, dupe := range group.Duplicates {
			match := group.Matches[j]
			g.Duplicates = append(g.Duplicates, Match{
				Path:       dupe.Path,
				Size:       dupe.Size,
				Percentage: match.Percentage,
			})
		}

		result.Groups = append(result.Groups, g)
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

// outputCSV outputs results in CSV format
func outputCSV(groups []*engine.DuplicateGroup, totalDupes int, totalSize int64, scanTime time.Duration) error {
	// Print header
	fmt.Println("group,type,path,size,match_percentage")

	// Print summary as comments
	fmt.Printf("# Scan completed in %s\n", scanTime)
	fmt.Printf("# Found %d duplicate groups with %d total duplicates\n", len(groups), totalDupes)
	fmt.Printf("# Total space that could be freed: %s\n", formatSize(totalSize))

	// Print data
	for i, group := range groups {
		// Print reference
		fmt.Printf("%d,reference,%s,%d,100\n", i+1, escapeCsvField(group.Reference.Path), group.Reference.Size)

		// Print duplicates
		for j, dupe := range group.Duplicates {
			match := group.Matches[j]
			fmt.Printf("%d,duplicate,%s,%d,%d\n", i+1, escapeCsvField(dupe.Path), dupe.Size, match.Percentage)
		}
	}

	return nil
}

// escapeCsvField escapes a field for CSV output
func escapeCsvField(field string) string {
	if strings.Contains(field, ",") || strings.Contains(field, "\"") || strings.Contains(field, "\n") {
		field = strings.ReplaceAll(field, "\"", "\"\"")
		field = "\"" + field + "\""
	}
	return field
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
