// code-from-spec: ROOT/tech_design/internal/spec_comment@v10
package speccomment

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SpecComment holds the logical name and version extracted from a
// generated file's spec reference comment.
type SpecComment struct {
	LogicalName string
	Version     int
}

// marker is the substring we search for on each line.
const marker = "code-from-spec: "

// ParseSpecComment reads filePath line by line from the top, looking
// for the spec comment marker. It stops as soon as a match is found.
// If no match is found, or the match is malformed, an error is returned.
func ParseSpecComment(filePath string) (*SpecComment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for the marker substring anywhere in the line.
		idx := strings.Index(line, marker)
		if idx < 0 {
			continue
		}

		// Extract everything after the marker to end of line.
		after := line[idx+len(marker):]

		// Trim trailing whitespace so trailing spaces don't interfere.
		after = strings.TrimRight(after, " \t\r")

		// Find the last occurrence of "@v" to split name and version.
		atIdx := strings.LastIndex(after, "@v")
		if atIdx < 0 {
			return nil, fmt.Errorf("malformed spec comment in %s: missing @v separator", filePath)
		}

		logicalName := after[:atIdx]
		if logicalName == "" {
			return nil, fmt.Errorf("malformed spec comment in %s: empty logical name", filePath)
		}

		// The version string is everything after "@v", up to the next
		// whitespace or end of string.
		versionStr := after[atIdx+2:]
		// Truncate at first whitespace if any.
		if spIdx := strings.IndexAny(versionStr, " \t"); spIdx >= 0 {
			versionStr = versionStr[:spIdx]
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf("malformed spec comment in %s: version %q is not a valid integer", filePath, versionStr)
		}

		return &SpecComment{
			LogicalName: logicalName,
			Version:     version,
		}, nil
	}

	// Check for scanner errors (e.g., I/O failure mid-read).
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	// Entire file scanned with no match.
	return nil, fmt.Errorf("no spec comment found in %s", filePath)
}
