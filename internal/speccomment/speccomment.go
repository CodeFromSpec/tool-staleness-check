// code-from-spec: ROOT/tech_design/internal/spec_comment@v12
// spec: ROOT/tech_design/internal/spec_comment@v12
//
// Package speccomment extracts the spec reference comment from generated
// source files for code staleness verification.
//
// The spec comment format is:
//
//	<comment-prefix> code-from-spec: <logical-name>@v<version>
//
// This package is language-agnostic: it scans each line for the marker
// substring regardless of the surrounding comment syntax.
package speccomment

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrNoSpecComment is returned (wrapped) when the file is fully read
// without finding a spec comment line.
// Callers use errors.Is(err, ErrNoSpecComment) to detect this case.
var ErrNoSpecComment = errors.New("no spec comment found")

// ErrMalformed is returned (wrapped) when a line containing the marker
// is found but the content after it cannot be parsed correctly.
// Callers use errors.Is(err, ErrMalformed) to detect this case.
var ErrMalformed = errors.New("malformed spec comment")

// SpecComment holds the logical name and version extracted from a
// generated file's spec reference comment.
type SpecComment struct {
	LogicalName string
	Version     int
}

// marker is the fixed substring we search for on every line.
const marker = "code-from-spec: "

// ParseSpecComment reads the file at filePath line by line from the top,
// looking for a line that contains the marker substring. It stops as soon
// as a match is found (efficiency: no state accumulated across lines).
//
// Return values:
//   - (*SpecComment, nil)           on success
//   - (nil, I/O error)              if the file cannot be opened or read
//   - (nil, ErrNoSpecComment-wrapped) if the whole file contains no marker
//   - (nil, ErrMalformed-wrapped)   if the marker is found but the payload
//     cannot be parsed
func ParseSpecComment(filePath string) (*SpecComment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		// I/O failure: wrap with path so callers get full context.
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer f.Close() //nolint:errcheck // Close on read-only file; error is irrelevant.

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// Search for the marker anywhere in the line.
		// This makes the parser language-agnostic: // # /* */ -- all work.
		idx := strings.Index(line, marker)
		if idx < 0 {
			// Not on this line — keep scanning.
			continue
		}

		// Take everything after the marker to the end of the line.
		after := line[idx+len(marker):]

		// Trim trailing whitespace so trailing spaces don't corrupt parsing.
		after = strings.TrimRight(after, " \t\r")

		// Find the LAST occurrence of "@v" to split logical name from version.
		// Using the last occurrence handles logical names that themselves
		// contain "@" (unlikely but defensive).
		atIdx := strings.LastIndex(after, "@v")
		if atIdx < 0 {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: missing @v separator: %w",
				filePath, ErrMalformed,
			)
		}

		logicalName := after[:atIdx]
		if logicalName == "" {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: empty logical name: %w",
				filePath, ErrMalformed,
			)
		}

		// The version string is everything after "@v", up to the next
		// whitespace character or end of string.
		versionStr := after[atIdx+2:]
		if spIdx := strings.IndexAny(versionStr, " \t"); spIdx >= 0 {
			versionStr = versionStr[:spIdx]
		}

		if versionStr == "" {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: empty version string: %w",
				filePath, ErrMalformed,
			)
		}

		version, err := strconv.Atoi(versionStr)
		if err != nil {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: version %q is not a valid integer: %w",
				filePath, versionStr, ErrMalformed,
			)
		}

		return &SpecComment{
			LogicalName: logicalName,
			Version:     version,
		}, nil
	}

	// Check for scanner-level I/O errors that occurred mid-read.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	// Entire file was read; no line contained the marker.
	return nil, fmt.Errorf("no spec comment found in %s: %w", filePath, ErrNoSpecComment)
}
