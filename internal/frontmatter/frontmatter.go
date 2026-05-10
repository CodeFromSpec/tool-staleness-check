// code-from-spec: ROOT/tech_design/internal/frontmatter@v13
package frontmatter

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// DependsOn represents a single cross-tree dependency declared in
// a node's frontmatter. Path is the logical name of the dependency
// and Version is the version number the node was written against.
type DependsOn struct {
	Path    string
	Version int
}

// Frontmatter holds all fields extracted from a node file's YAML
// frontmatter block plus the title line that follows it.
//
// Pointer fields (Version, ParentVersion, SubjectVersion) are nil
// when the corresponding YAML key is absent. Validation of which
// fields are required is the caller's responsibility — the parser
// treats every field as optional.
type Frontmatter struct {
	Version        *int
	ParentVersion  *int
	SubjectVersion *int
	DependsOn      []DependsOn
	Implements     []string
	Title          string
}

// rawFrontmatter is the intermediate YAML-deserialization target.
// Field names use yaml struct tags to match the on-disk format.
// Unknown fields are silently ignored because we do not use
// "strict" decoding.
type rawFrontmatter struct {
	Version        *int `yaml:"version"`
	ParentVersion  *int `yaml:"parent_version"`
	SubjectVersion *int `yaml:"subject_version"`
	DependsOn      []struct {
		Path    string `yaml:"path"`
		Version int    `yaml:"version"`
	} `yaml:"depends_on"`
	Implements []string `yaml:"implements"`
}

// ParseFrontmatter reads the file at filePath, extracts the YAML
// frontmatter block and the title line, and returns a Frontmatter
// struct. It reads line by line and stops as soon as it has both
// the frontmatter and the title — the rest of the file is never
// read. Caching is the caller's responsibility.
//
// The frontmatter is expected to be a YAML block between the first
// "---" and the second "---" at the top of the file. The title is
// the first non-empty line after the closing "---", expected to
// start with "# ". If the title line is missing or does not start
// with "# ", Title is stored as empty string.
//
// Error messages follow the form:
//   - "error reading <path>: <underlying error>"
//   - "error parsing frontmatter in <path>: <underlying error>"
//   - "frontmatter not found in <path>"
func ParseFrontmatter(filePath string) (*Frontmatter, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// --- State machine ---
	// Phase 1: find the opening "---"
	// Phase 2: collect YAML lines until the closing "---"
	// Phase 3: find the first non-empty line (title)

	// Phase 1: locate the opening delimiter.
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

	// Phase 2: collect YAML lines until the closing delimiter.
	// Only these intermediate lines are buffered; once parsed they
	// are discarded (spec requirement: intermediate state discarded
	// after parsing).
	var yamlLines []string
	foundClosing := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundClosing = true
			break
		}
		yamlLines = append(yamlLines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}
	if !foundClosing {
		return nil, fmt.Errorf("frontmatter not found in %s", filePath)
	}

	// Parse the collected YAML lines into rawFrontmatter.
	// yamlLines is the only intermediate state and is not retained
	// beyond this point.
	yamlContent := strings.Join(yamlLines, "\n")
	var raw rawFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, fmt.Errorf("error parsing frontmatter in %s: %w", filePath, err)
	}

	// Build the exported Frontmatter struct from the raw target.
	fm := &Frontmatter{
		Version:        raw.Version,
		ParentVersion:  raw.ParentVersion,
		SubjectVersion: raw.SubjectVersion,
		Implements:     raw.Implements,
	}

	// Convert raw DependsOn entries to exported DependsOn structs.
	for _, d := range raw.DependsOn {
		fm.DependsOn = append(fm.DependsOn, DependsOn{
			Path:    d.Path,
			Version: d.Version,
		})
	}

	// Phase 3: extract the title — the first non-empty line after
	// the closing "---". If it starts with "# ", store the text
	// after that prefix. Otherwise store an empty string and stop.
	// Either way we stop after the first non-empty line, so the
	// rest of the file body is never read.
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Skip blank lines between closing "---" and title.
			continue
		}
		// First non-empty line found — this is the candidate title.
		if strings.HasPrefix(trimmed, "# ") {
			fm.Title = strings.TrimPrefix(trimmed, "# ")
		}
		// Whether or not it matched "# ", we are done reading.
		break
	}
	// A scanner error here is still a read error.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filePath, err)
	}

	return fm, nil
}
