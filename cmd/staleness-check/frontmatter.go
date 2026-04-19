// spec: ROOT/tech_design/frontmatter@v6
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DependsOn represents a cross-tree dependency with its known version.
// Corresponds to an entry in the frontmatter `depends_on` list.
type DependsOn struct {
	Path    string `yaml:"path"`
	Version int    `yaml:"version"`
}

// Frontmatter holds the parsed YAML frontmatter and title extracted from
// a node file. All fields are optional at the parsing level — validation
// of required fields happens during staleness verification.
type Frontmatter struct {
	Version       *int       `yaml:"version"`
	ParentVersion *int       `yaml:"parent_version"`
	DependsOn     []DependsOn `yaml:"depends_on"`
	Implements    []string   `yaml:"implements"`
	// Title is the text after "# " on the first non-empty line after the
	// frontmatter closing "---". Empty string if missing or malformed.
	Title string `yaml:"-"`
}

// ParseFrontmatter reads a node file at filePath, extracts the YAML
// frontmatter and title, and returns the result. It reads line by line
// and stops as soon as both frontmatter and title are extracted — the
// rest of the file is never read. It does not cache; caching is the
// caller's responsibility.
func ParseFrontmatter(filePath string) (fm *Frontmatter, err error) {
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

	// --- Phase 1: Find the opening "---" delimiter. ---
	foundOpening := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundOpening = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	if !foundOpening {
		return nil, fmt.Errorf("frontmatter not found in %s", filePath)
	}

	// --- Phase 2: Collect raw frontmatter lines until the closing "---". ---
	var rawLines []string
	foundClosing := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClosing = true
			break
		}
		rawLines = append(rawLines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	if !foundClosing {
		return nil, fmt.Errorf("frontmatter not found in %s", filePath)
	}

	// --- Phase 3: Parse the collected YAML into a Frontmatter struct. ---
	// Join the raw lines into a single YAML document and discard them
	// immediately after parsing to avoid retaining intermediate state.
	rawYAML := strings.Join(rawLines, "\n")
	rawLines = nil // discard intermediate state

	fm = &Frontmatter{}
	if err := yaml.Unmarshal([]byte(rawYAML), fm); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter in %s: %w", filePath, err)
	}

	// --- Phase 4: Extract the title from the first non-empty line after "---". ---
	// The title is expected to start with "# ". If missing or malformed,
	// store empty string — the caller decides how to handle it.
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Skip empty lines between closing "---" and the title.
			continue
		}
		// First non-empty line found — check for title format.
		if strings.HasPrefix(trimmed, "# ") {
			fm.Title = strings.TrimPrefix(trimmed, "# ")
		}
		// Whether or not it matched, stop reading — we only care about
		// the first non-empty line.
		break
	}
	// Check for scanner errors after the title extraction loop.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	return fm, nil
}
