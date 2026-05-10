// code-from-spec: ROOT/tech_design/internal/spec_comment@v14
//
// Package speccomment extracts the spec reference comment from generated
// source files for code staleness verification.
//
// The spec comment format is:
//
//	<comment-prefix> code-from-spec: <logical-name>@v<version>
//
// This package is language-agnostic: it scans each line for the marker
// substring regardless of the surrounding comment syntax. Any comment
// style works — //, #, /* */, --, etc.
package speccomment

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrNoSpecComment is returned (wrapped) when the entire file is read
// without finding a line containing the spec comment marker.
// Callers use errors.Is(err, ErrNoSpecComment) to detect this case.
var ErrNoSpecComment = errors.New("no spec comment found")

// ErrMalformed is returned (wrapped) when a line containing the marker
// is found but the content after it cannot be parsed correctly — for
// example, missing @v separator, empty logical name, or non-integer version.
// Callers use errors.Is(err, ErrMalformed) to detect this case.
var ErrMalformed = errors.New("malformed spec comment")

// SpecComment holds the parsed result of a spec reference comment found
// in a generated source file.
type SpecComment struct {
	// LogicalName is the spec node name embedded in the comment,
	// e.g. "ROOT/architecture/backend/config".
	LogicalName string

	// Version is the integer version number embedded in the comment,
	// e.g. 5 from "@v5".
	Version int
}

// marker is the fixed substring we search for on every line. Its presence
// indicates a spec comment regardless of what precedes it (comment prefix).
const marker = "code-from-spec: "

// ParseSpecComment reads the file at filePath line by line from the top,
// scanning for a line that contains the marker substring.
//
// It stops as soon as a match is found — no state is accumulated across
// lines, satisfying the efficiency requirement.
//
// Extraction algorithm (per spec):
//  1. Take everything after "code-from-spec: " to the end of the line.
//  2. Find the LAST occurrence of "@v" in that substring.
//  3. Everything before "@v" is the logical name.
//  4. Everything after "@v", up to the next whitespace or end of line,
//     is the version string.
//  5. Parse the version string as an integer.
//
// Return values:
//   - (*SpecComment, nil)               on success
//   - (nil, I/O-error)                  if the file cannot be opened or read
//   - (nil, ErrNoSpecComment-wrapped)   if the entire file contains no marker
//   - (nil, ErrMalformed-wrapped)       if the marker is found but the payload
//     is not parseable (missing @v, empty logical name, bad version integer)
func ParseSpecComment(filePath string) (*SpecComment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		// I/O failure: include the path so callers get actionable context.
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer f.Close() //nolint:errcheck // Read-only open; close error is not actionable.

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		// Search for the marker anywhere on the line. This approach is
		// intentionally language-agnostic: the comment prefix (// # --
		// etc.) is irrelevant — we only care about what follows the marker.
		idx := strings.Index(line, marker)
		if idx < 0 {
			// Marker not found on this line; keep scanning.
			continue
		}

		// Extract everything after "code-from-spec: " to end of line.
		after := line[idx+len(marker):]

		// Trim trailing whitespace to avoid corrupting version parsing.
		after = strings.TrimRight(after, " \t\r")

		// Step 2: Find the LAST occurrence of "@v". Using the last occurrence
		// is defensive against logical names that might themselves contain "@".
		atIdx := strings.LastIndex(after, "@v")
		if atIdx < 0 {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: missing @v separator: %w",
				filePath, ErrMalformed,
			)
		}

		// Step 3: Logical name is everything before "@v".
		logicalName := after[:atIdx]
		if logicalName == "" {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: empty logical name: %w",
				filePath, ErrMalformed,
			)
		}

		// Step 4: Version string is everything after "@v", up to the next
		// whitespace or end of string. The "+2" skips past the literal "@v".
		versionStr := after[atIdx+2:]
		if spIdx := strings.IndexAny(versionStr, " \t"); spIdx >= 0 {
			// Trim any trailing content (e.g., extra comment text).
			versionStr = versionStr[:spIdx]
		}

		if versionStr == "" {
			return nil, fmt.Errorf(
				"malformed spec comment in %s: empty version string: %w",
				filePath, ErrMalformed,
			)
		}

		// Step 5: Parse version string as an integer.
		version, parseErr := strconv.Atoi(versionStr)
		if parseErr != nil {
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

	// Check for any scanner-level I/O error that occurred during scanning
	// (distinct from os.Open failure above).
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, scanErr)
	}

	// The entire file was read without finding the marker.
	return nil, fmt.Errorf("no spec comment found in %s: %w", filePath, ErrNoSpecComment)
}
