// spec: ROOT/tech_design/spec_comment@v6
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SpecComment holds the logical name and version extracted from a
// spec reference comment found in a generated source file.
// See ROOT/domain/specifications for the comment format definition.
type SpecComment struct {
	LogicalName string
	Version     int
}

// ParseSpecComment reads filePath line by line looking for the spec
// reference comment pattern "spec: <logical-name>@v<version>". It
// stops reading as soon as a match is found — the rest of the file
// is never read. No intermediate state is accumulated across lines.
//
// Returns the parsed SpecComment on success, or an error describing
// what went wrong: file not readable, no spec comment found, or
// malformed spec comment.
func ParseSpecComment(filePath string) (sc *SpecComment, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("error closing %s: %w", filePath, cerr)
		}
	}()

	scanner := bufio.NewScanner(f)

	// Scan line by line looking for the "spec: " substring.
	// Stop on first match — no state is retained from previous lines.
	for scanner.Scan() {
		line := scanner.Text()

		// Look for the "spec: " marker anywhere in the line.
		// This is language-agnostic: the tool does not attempt to
		// identify the comment syntax of the language.
		idx := strings.Index(line, "spec: ")
		if idx < 0 {
			continue
		}

		// Found the spec comment line — extract the payload after "spec: ".
		payload := line[idx+len("spec: "):]

		// Trim trailing whitespace so end-of-line spaces don't interfere.
		payload = strings.TrimRight(payload, " \t\r")

		// Find the last occurrence of "@v" to split logical name and version.
		// Using the last occurrence handles logical names that might contain "@v".
		atIdx := strings.LastIndex(payload, "@v")
		if atIdx < 0 {
			return nil, fmt.Errorf("malformed spec comment in %s: missing @v separator", filePath)
		}

		logicalName := payload[:atIdx]
		if logicalName == "" {
			return nil, fmt.Errorf("malformed spec comment in %s: empty logical name", filePath)
		}

		// The version string is everything after "@v", up to the next
		// whitespace or end of the payload.
		versionStr := payload[atIdx+len("@v"):]
		// Truncate at the first whitespace if any remains after trimming.
		if spaceIdx := strings.IndexAny(versionStr, " \t"); spaceIdx >= 0 {
			versionStr = versionStr[:spaceIdx]
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

	// Check for scanner errors after exhausting all lines.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	// Entire file scanned without finding a spec comment.
	return nil, fmt.Errorf("no spec comment found in %s", filePath)
}
