// spec: TEST/tech_design/frontmatter@v1
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile creates a file in dir with the given content and returns
// the full path. This helper keeps each test self-contained — the file
// is written to t.TempDir() so cleanup is automatic.
func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file %s: %v", name, err)
	}
	return p
}

// --- Happy Path ---

// TestParseFrontmatter_CompleteFrontmatter verifies that all frontmatter
// fields are parsed correctly when every field is present, including
// depends_on with multiple entries and implements with multiple paths.
func TestParseFrontmatter_CompleteFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 3\nparent_version: 2\ndepends_on:\n  - path: ROOT/other\n    version: 1\n  - path: EXTERNAL/database\n    version: 5\nimplements:\n  - internal/config/config.go\n  - internal/config/config_test.go\n---\n\n# ROOT/some/node\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version == nil || *fm.Version != 3 {
		t.Errorf("Version = %v, want 3", fm.Version)
	}
	if fm.ParentVersion == nil || *fm.ParentVersion != 2 {
		t.Errorf("ParentVersion = %v, want 2", fm.ParentVersion)
	}

	// Verify depends_on entries.
	if len(fm.DependsOn) != 2 {
		t.Fatalf("len(DependsOn) = %d, want 2", len(fm.DependsOn))
	}
	if fm.DependsOn[0].Path != "ROOT/other" || fm.DependsOn[0].Version != 1 {
		t.Errorf("DependsOn[0] = %+v, want {Path:ROOT/other Version:1}", fm.DependsOn[0])
	}
	if fm.DependsOn[1].Path != "EXTERNAL/database" || fm.DependsOn[1].Version != 5 {
		t.Errorf("DependsOn[1] = %+v, want {Path:EXTERNAL/database Version:5}", fm.DependsOn[1])
	}

	// Verify implements entries.
	if len(fm.Implements) != 2 {
		t.Fatalf("len(Implements) = %d, want 2", len(fm.Implements))
	}
	if fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("Implements[0] = %q, want %q", fm.Implements[0], "internal/config/config.go")
	}
	if fm.Implements[1] != "internal/config/config_test.go" {
		t.Errorf("Implements[1] = %q, want %q", fm.Implements[1], "internal/config/config_test.go")
	}

	// Verify title extraction.
	if fm.Title != "ROOT/some/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/some/node")
	}
}

// TestParseFrontmatter_RootNode verifies parsing of a root node that has
// only version and no parent_version, depends_on, or implements.
func TestParseFrontmatter_RootNode(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 5\n---\n\n# ROOT\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version == nil || *fm.Version != 5 {
		t.Errorf("Version = %v, want 5", fm.Version)
	}
	if fm.ParentVersion != nil {
		t.Errorf("ParentVersion = %v, want nil (absent)", fm.ParentVersion)
	}
	if fm.DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements = %v, want nil", fm.Implements)
	}
	if fm.Title != "ROOT" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT")
	}
}

// TestParseFrontmatter_ExternalDependency verifies parsing of an
// external dependency file that has only version.
func TestParseFrontmatter_ExternalDependency(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 2\n---\n\n# EXTERNAL/database\n"
	p := writeTestFile(t, dir, "external.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version == nil || *fm.Version != 2 {
		t.Errorf("Version = %v, want 2", fm.Version)
	}
	if fm.Title != "EXTERNAL/database" {
		t.Errorf("Title = %q, want %q", fm.Title, "EXTERNAL/database")
	}
}

// TestParseFrontmatter_IgnoresUnknownFields verifies that unknown YAML
// fields in the frontmatter are silently ignored and do not cause errors.
func TestParseFrontmatter_IgnoresUnknownFields(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\nparent_version: 1\nsome_future_field: hello\nanother: 42\n---\n\n# ROOT/node\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version == nil || *fm.Version != 1 {
		t.Errorf("Version = %v, want 1", fm.Version)
	}
	if fm.ParentVersion == nil || *fm.ParentVersion != 1 {
		t.Errorf("ParentVersion = %v, want 1", fm.ParentVersion)
	}
}

// TestParseFrontmatter_TestNodeTitle verifies that a TEST/ title is
// correctly extracted.
func TestParseFrontmatter_TestNodeTitle(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\nparent_version: 2\nimplements:\n  - internal/config/config_test.go\n---\n\n# TEST/some/node\n"
	p := writeTestFile(t, dir, "test.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "TEST/some/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "TEST/some/node")
	}
}

// TestParseFrontmatter_NamedTestNodeTitle verifies that a named test
// node title with parenthesized name is correctly extracted.
func TestParseFrontmatter_NamedTestNodeTitle(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\nparent_version: 2\nimplements:\n  - internal/config/config_edge_test.go\n---\n\n# TEST/some/node(edge_cases)\n"
	p := writeTestFile(t, dir, "test.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "TEST/some/node(edge_cases)" {
		t.Errorf("Title = %q, want %q", fm.Title, "TEST/some/node(edge_cases)")
	}
}

// --- Edge Cases ---

// TestParseFrontmatter_EmptyFrontmatter verifies that an empty
// frontmatter block (no fields between delimiters) produces zero values
// for all fields and no error.
func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := "---\n---\n\n# ROOT/node\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version != nil {
		t.Errorf("Version = %v, want nil (absent)", fm.Version)
	}
	if fm.ParentVersion != nil {
		t.Errorf("ParentVersion = %v, want nil (absent)", fm.ParentVersion)
	}
	if fm.DependsOn != nil {
		t.Errorf("DependsOn = %v, want nil", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("Implements = %v, want nil", fm.Implements)
	}
	if fm.Title != "ROOT/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/node")
	}
}

// TestParseFrontmatter_BlankLinesBetweenFrontmatterAndTitle verifies
// that blank lines after the closing "---" are skipped when searching
// for the title.
func TestParseFrontmatter_BlankLinesBetweenFrontmatterAndTitle(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\n---\n\n\n# ROOT/node\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "ROOT/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/node")
	}
}

// TestParseFrontmatter_NoTitleLine verifies that when the first
// non-empty line after frontmatter is not a "# " title, the Title
// field is empty and no error is returned.
func TestParseFrontmatter_NoTitleLine(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\n---\n\nSome text without a title.\n"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "" {
		t.Errorf("Title = %q, want empty string", fm.Title)
	}
}

// TestParseFrontmatter_OnlyFrontmatterNothingAfter verifies that a
// file ending immediately after the closing "---" produces an empty
// title and no error.
func TestParseFrontmatter_OnlyFrontmatterNothingAfter(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: 1\n---"
	p := writeTestFile(t, dir, "node.md", content)

	fm, err := ParseFrontmatter(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "" {
		t.Errorf("Title = %q, want empty string", fm.Title)
	}
}

// --- Failure Cases ---

// TestParseFrontmatter_FileDoesNotExist verifies that calling
// ParseFrontmatter with a non-existent path returns an error whose
// message contains the file path.
func TestParseFrontmatter_FileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "does_not_exist.md")

	_, err := ParseFrontmatter(fakePath)
	if err == nil {
		t.Fatal("expected an error for non-existent file, got nil")
	}

	// The error message must contain the file path so the user knows
	// which file failed.
	if !strings.Contains(err.Error(), fakePath) {
		t.Errorf("error should contain file path %q, got: %s", fakePath, err.Error())
	}
}

// TestParseFrontmatter_NoFrontmatterDelimiters verifies that a file
// with no "---" delimiters at all returns an error indicating
// frontmatter was not found.
func TestParseFrontmatter_NoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	content := "Just some text.\n"
	p := writeTestFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(p)
	if err == nil {
		t.Fatal("expected an error for missing frontmatter, got nil")
	}

	// The error should indicate frontmatter was not found.
	if !strings.Contains(err.Error(), "frontmatter not found") {
		t.Errorf("error should contain %q, got: %s", "frontmatter not found", err.Error())
	}
}

// TestParseFrontmatter_MalformedYAML verifies that invalid YAML between
// frontmatter delimiters returns an error indicating a parse failure.
func TestParseFrontmatter_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	content := "---\nversion: [invalid\n---\n"
	p := writeTestFile(t, dir, "node.md", content)

	_, err := ParseFrontmatter(p)
	if err == nil {
		t.Fatal("expected an error for malformed YAML, got nil")
	}

	// The error should indicate a parsing failure.
	if !strings.Contains(err.Error(), "error parsing frontmatter") {
		t.Errorf("error should contain %q, got: %s", "error parsing frontmatter", err.Error())
	}
}
