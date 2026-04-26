// code-from-spec: TEST/tech_design/internal/spec_comment@v7
package speccomment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestGoStyleComment verifies parsing of a Go-style // comment.
func TestGoStyleComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.go")
	content := "// code-from-spec: ROOT/architecture/backend/config@v5\npackage configuration\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/architecture/backend/config", 5)
}

// TestPythonStyleComment verifies parsing of a Python-style # comment.
func TestPythonStyleComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "staleness.py")
	content := "# code-from-spec: ROOT/domain/staleness@v3\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/domain/staleness", 3)
}

// TestHTMLStyleComment verifies parsing of an HTML-style <!-- --> comment.
func TestHTMLStyleComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "template.html")
	content := "<!-- code-from-spec: ROOT/frontend/template@v2 -->\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/frontend/template", 2)
}

// TestBlockCommentSingleLine verifies parsing of a C-style /* */ comment.
func TestBlockCommentSingleLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.c")
	content := "/* code-from-spec: ROOT/some/node@v1 */\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/some/node", 1)
}

// TestTestNodeCanonical verifies parsing of a TEST/ logical name.
func TestTestNodeCanonical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config_test.go")
	content := "// code-from-spec: TEST/domain/config@v3\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "TEST/domain/config", 3)
}

// TestTestNodeNamed verifies parsing of a TEST/ logical name with a
// parenthesized variant name.
func TestTestNodeNamed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config_edge_test.go")
	content := "// code-from-spec: TEST/domain/config(edge_cases)@v2\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "TEST/domain/config(edge_cases)", 2)
}

// TestCommentNotOnFirstLine verifies that the spec comment is found even
// when preceded by a shebang and license header.
func TestCommentNotOnFirstLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deploy.py")
	content := "#!/usr/bin/env python3\n# License: MIT\n# code-from-spec: ROOT/scripts/deploy@v1\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/scripts/deploy", 1)
}

// TestSQLStyleComment verifies parsing of a SQL-style -- comment.
func TestSQLStyleComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrations.sql")
	content := "-- code-from-spec: ROOT/database/migrations@v4\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/database/migrations", 4)
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

// TestSpecCommentDeepInFile verifies that the spec comment is found even
// after 100 lines of other code.
func TestSpecCommentDeepInFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deep.go")

	var sb strings.Builder
	// Write 100 lines of filler code before the spec comment.
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("// line %d\n", i+1))
	}
	sb.WriteString("// code-from-spec: ROOT/deep/node@v7\n")
	writeTestFile(t, path, sb.String())

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/deep/node", 7)
}

// TestTrailingWhitespaceAfterVersion verifies that trailing whitespace
// after the version number is ignored.
func TestTrailingWhitespaceAfterVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trailing.go")
	// Three trailing spaces after the version number.
	content := "// code-from-spec: ROOT/node@v3   \n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/node", 3)
}

// TestLogicalNameWithManySegments verifies that deeply nested logical
// names are parsed correctly.
func TestLogicalNameWithManySegments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "many.go")
	content := "// code-from-spec: ROOT/a/b/c/d/e/f@v1\n"
	writeTestFile(t, path, content)

	sc, err := ParseSpecComment(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectSpecComment(t, sc, "ROOT/a/b/c/d/e/f", 1)
}

// ---------------------------------------------------------------------------
// Failure Cases
// ---------------------------------------------------------------------------

// TestFileDoesNotExist verifies that a non-existent path produces an error
// that contains the file path.
func TestFileDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.go")

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error should contain file path %q, got: %v", path, err)
	}
}

// TestNoSpecCommentInFile verifies that a file without any spec comment
// produces an appropriate error.
func TestNoSpecCommentInFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noop.go")
	content := "package main\n\nfunc main() {}\n"
	writeTestFile(t, path, content)

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for missing spec comment, got nil")
	}
	if !strings.Contains(err.Error(), "no spec comment") {
		t.Fatalf("error should indicate no spec comment found, got: %v", err)
	}
}

// TestMissingVersion verifies that "code-from-spec: ROOT/node@v" (empty
// version) is treated as malformed.
func TestMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nover.go")
	content := "// code-from-spec: ROOT/node@v\n"
	writeTestFile(t, path, content)

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("error should indicate malformed spec comment, got: %v", err)
	}
}

// TestMissingAtVSeparator verifies that a spec comment without @v is
// treated as malformed.
func TestMissingAtVSeparator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nosep.go")
	content := "// code-from-spec: ROOT/node\n"
	writeTestFile(t, path, content)

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for missing @v separator, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("error should indicate malformed spec comment, got: %v", err)
	}
}

// TestNonIntegerVersion verifies that a non-numeric version string is
// treated as malformed.
func TestNonIntegerVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badver.go")
	content := "// code-from-spec: ROOT/node@vabc\n"
	writeTestFile(t, path, content)

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for non-integer version, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("error should indicate malformed spec comment, got: %v", err)
	}
}

// TestEmptyLogicalName verifies that an empty logical name (just @v with
// a version) is treated as malformed.
func TestEmptyLogicalName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.go")
	content := "// code-from-spec: @v3\n"
	writeTestFile(t, path, content)

	_, err := ParseSpecComment(path)
	if err == nil {
		t.Fatal("expected error for empty logical name, got nil")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Fatalf("error should indicate malformed spec comment, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeTestFile creates a file at path with the given content. It fails
// the test immediately if the file cannot be written.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
}

// expectSpecComment asserts that sc has the expected logical name and version.
func expectSpecComment(t *testing.T, sc *SpecComment, wantName string, wantVersion int) {
	t.Helper()
	if sc.LogicalName != wantName {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, wantName)
	}
	if sc.Version != wantVersion {
		t.Errorf("Version = %d, want %d", sc.Version, wantVersion)
	}
}
