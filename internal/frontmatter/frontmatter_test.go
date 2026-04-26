// code-from-spec: TEST/tech_design/internal/frontmatter@v6
package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Happy Path ---

// TestParsesCompleteFrontmatter verifies that all frontmatter fields
// are correctly parsed from a file containing every supported field.
func TestParsesCompleteFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 3
parent_version: 2
depends_on:
  - path: ROOT/other
    version: 1
  - path: ROOT/another
    version: 5
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---

# ROOT/some/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Version
	assertIntPtr(t, "Version", fm.Version, 3)

	// ParentVersion
	assertIntPtr(t, "ParentVersion", fm.ParentVersion, 2)

	// SubjectVersion should be nil (not present)
	if fm.SubjectVersion != nil {
		t.Errorf("expected SubjectVersion to be nil, got %d", *fm.SubjectVersion)
	}

	// DependsOn
	if len(fm.DependsOn) != 2 {
		t.Fatalf("expected 2 DependsOn entries, got %d", len(fm.DependsOn))
	}
	if fm.DependsOn[0].Path != "ROOT/other" || fm.DependsOn[0].Version != 1 {
		t.Errorf("DependsOn[0] = %+v, want {Path:ROOT/other Version:1}", fm.DependsOn[0])
	}
	if fm.DependsOn[1].Path != "ROOT/another" || fm.DependsOn[1].Version != 5 {
		t.Errorf("DependsOn[1] = %+v, want {Path:ROOT/another Version:5}", fm.DependsOn[1])
	}

	// Implements
	if len(fm.Implements) != 2 {
		t.Fatalf("expected 2 Implements entries, got %d", len(fm.Implements))
	}
	if fm.Implements[0] != "internal/config/config.go" {
		t.Errorf("Implements[0] = %q, want %q", fm.Implements[0], "internal/config/config.go")
	}
	if fm.Implements[1] != "internal/config/config_test.go" {
		t.Errorf("Implements[1] = %q, want %q", fm.Implements[1], "internal/config/config_test.go")
	}

	// Title
	if fm.Title != "ROOT/some/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/some/node")
	}
}

// TestParsesRootNode verifies parsing a root node that has only
// version and no parent_version.
func TestParsesRootNode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 5
---

# ROOT
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIntPtr(t, "Version", fm.Version, 5)

	if fm.ParentVersion != nil {
		t.Errorf("expected ParentVersion to be nil, got %d", *fm.ParentVersion)
	}
	if fm.SubjectVersion != nil {
		t.Errorf("expected SubjectVersion to be nil, got %d", *fm.SubjectVersion)
	}
	if fm.DependsOn != nil {
		t.Errorf("expected DependsOn to be nil, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected Implements to be nil, got %v", fm.Implements)
	}
	if fm.Title != "ROOT" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT")
	}
}

// TestParsesTestNodeWithSubjectVersion verifies parsing a test node
// that has subject_version instead of parent_version.
func TestParsesTestNodeWithSubjectVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default.test.md")

	content := `---
version: 2
subject_version: 5
implements:
  - internal/config/config_test.go
---

# TEST/some/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIntPtr(t, "Version", fm.Version, 2)
	assertIntPtr(t, "SubjectVersion", fm.SubjectVersion, 5)

	if fm.ParentVersion != nil {
		t.Errorf("expected ParentVersion to be nil, got %d", *fm.ParentVersion)
	}
	if fm.Title != "TEST/some/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "TEST/some/node")
	}
}

// TestParsesExternalDependency verifies parsing a simple node with
// only version (like an external dependency node).
func TestParsesExternalDependency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 2
---

# ROOT/external/database
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIntPtr(t, "Version", fm.Version, 2)
	if fm.Title != "ROOT/external/database" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/external/database")
	}
}

// TestIgnoresUnknownFrontmatterFields verifies that unknown YAML
// fields in the frontmatter are silently ignored.
func TestIgnoresUnknownFrontmatterFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---

# ROOT/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIntPtr(t, "Version", fm.Version, 1)
	assertIntPtr(t, "ParentVersion", fm.ParentVersion, 1)
	if fm.Title != "ROOT/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/node")
	}
}

// TestParsesTestNodeTitle verifies that a test node title starting
// with TEST/ is extracted correctly.
func TestParsesTestNodeTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default.test.md")

	content := `---
version: 1
subject_version: 2
implements:
  - internal/config/config_test.go
---

# TEST/some/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "TEST/some/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "TEST/some/node")
	}
}

// TestParsesNamedTestNodeTitle verifies that a named test node title
// (with parenthetical name) is extracted correctly.
func TestParsesNamedTestNodeTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edge_cases.test.md")

	content := `---
version: 1
subject_version: 2
implements:
  - internal/config/config_edge_test.go
---

# TEST/some/node(edge_cases)
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "TEST/some/node(edge_cases)" {
		t.Errorf("Title = %q, want %q", fm.Title, "TEST/some/node(edge_cases)")
	}
}

// --- Edge Cases ---

// TestEmptyFrontmatter verifies that an empty frontmatter block
// (no fields between delimiters) results in all-nil/zero fields.
func TestEmptyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
---

# ROOT/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Version != nil {
		t.Errorf("expected Version to be nil, got %d", *fm.Version)
	}
	if fm.ParentVersion != nil {
		t.Errorf("expected ParentVersion to be nil, got %d", *fm.ParentVersion)
	}
	if fm.SubjectVersion != nil {
		t.Errorf("expected SubjectVersion to be nil, got %d", *fm.SubjectVersion)
	}
	if fm.DependsOn != nil {
		t.Errorf("expected DependsOn to be nil, got %v", fm.DependsOn)
	}
	if fm.Implements != nil {
		t.Errorf("expected Implements to be nil, got %v", fm.Implements)
	}
	if fm.Title != "ROOT/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/node")
	}
}

// TestTitleWithBlankLinesBetweenFrontmatterAndTitle verifies that
// blank lines between the closing --- and the title are skipped.
func TestTitleWithBlankLinesBetweenFrontmatterAndTitle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 1
---


# ROOT/node
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "ROOT/node" {
		t.Errorf("Title = %q, want %q", fm.Title, "ROOT/node")
	}
}

// TestNoTitleLine verifies that a file with frontmatter but no
// title line (no "# " prefix) results in an empty title.
func TestNoTitleLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: 1
---

Some text without a title.
`
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fm.Title != "" {
		t.Errorf("Title = %q, want empty string", fm.Title)
	}
}

// TestFileWithOnlyFrontmatter verifies that a file with frontmatter
// but nothing after the closing delimiter results in empty title.
func TestFileWithOnlyFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	// No trailing newline after closing ---
	content := "---\nversion: 1\n---"
	writeTestFile(t, path, content)

	fm, err := ParseFrontmatter(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertIntPtr(t, "Version", fm.Version, 1)
	if fm.Title != "" {
		t.Errorf("Title = %q, want empty string", fm.Title)
	}
}

// --- Failure Cases ---

// TestFileDoesNotExist verifies that a non-existent file produces
// an error containing the file path.
func TestFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.md")

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	// The error message must contain the file path.
	if !containsString(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

// TestNoFrontmatterDelimiters verifies that a file with no ---
// delimiters produces a "frontmatter not found" error.
func TestNoFrontmatterDelimiters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := "Just some text.\n"
	writeTestFile(t, path, content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}

	// The error must mention the file path.
	if !containsString(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

// TestMalformedYAMLInFrontmatter verifies that invalid YAML between
// frontmatter delimiters produces a parse error.
func TestMalformedYAMLInFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_node.md")

	content := `---
version: [invalid
---
`
	writeTestFile(t, path, content)

	_, err := ParseFrontmatter(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}

	// The error must mention the file path.
	if !containsString(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

// --- Helpers ---

// writeTestFile creates a file at the given path with the given content.
// It fails the test immediately if the file cannot be written.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
}

// assertIntPtr checks that a *int field has the expected value.
func assertIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %d", name, want)
	}
	if *got != want {
		t.Errorf("%s = %d, want %d", name, *got, want)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

// stringContains is a simple contains check without importing strings.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
