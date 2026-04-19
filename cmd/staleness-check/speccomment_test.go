// spec: TEST/tech_design/spec_comment@v1
package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// --- Happy Path ---

// TestParseSpecComment_GoStyleComment verifies that a Go-style "//" comment
// containing the spec reference is correctly parsed.
func TestParseSpecComment_GoStyleComment(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/architecture/backend/config@v5\npackage configuration\n"
	p := writeTestFile(t, dir, "config.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/architecture/backend/config" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/architecture/backend/config")
	}
	if sc.Version != 5 {
		t.Errorf("Version = %d, want 5", sc.Version)
	}
}

// TestParseSpecComment_PythonStyleComment verifies that a Python-style "#"
// comment containing the spec reference is correctly parsed.
func TestParseSpecComment_PythonStyleComment(t *testing.T) {
	dir := t.TempDir()
	content := "# spec: ROOT/domain/staleness@v3\n"
	p := writeTestFile(t, dir, "staleness.py", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/domain/staleness" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/domain/staleness")
	}
	if sc.Version != 3 {
		t.Errorf("Version = %d, want 3", sc.Version)
	}
}

// TestParseSpecComment_HTMLStyleComment verifies that an HTML-style "<!-- -->"
// comment containing the spec reference is correctly parsed.
func TestParseSpecComment_HTMLStyleComment(t *testing.T) {
	dir := t.TempDir()
	content := "<!-- spec: ROOT/frontend/template@v2 -->\n"
	p := writeTestFile(t, dir, "template.html", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/frontend/template" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/frontend/template")
	}
	if sc.Version != 2 {
		t.Errorf("Version = %d, want 2", sc.Version)
	}
}

// TestParseSpecComment_BlockCommentSingleLine verifies that a single-line
// block comment "/* */" containing the spec reference is correctly parsed.
func TestParseSpecComment_BlockCommentSingleLine(t *testing.T) {
	dir := t.TempDir()
	content := "/* spec: ROOT/some/node@v1 */\n"
	p := writeTestFile(t, dir, "node.c", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/some/node" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/some/node")
	}
	if sc.Version != 1 {
		t.Errorf("Version = %d, want 1", sc.Version)
	}
}

// TestParseSpecComment_TestNodeCanonical verifies that a canonical test node
// logical name (TEST/ prefix) is correctly parsed.
func TestParseSpecComment_TestNodeCanonical(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: TEST/domain/config@v3\n"
	p := writeTestFile(t, dir, "config_test.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "TEST/domain/config" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "TEST/domain/config")
	}
	if sc.Version != 3 {
		t.Errorf("Version = %d, want 3", sc.Version)
	}
}

// TestParseSpecComment_TestNodeNamed verifies that a named test node logical
// name with parenthesized name suffix is correctly parsed.
func TestParseSpecComment_TestNodeNamed(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: TEST/domain/config(edge_cases)@v2\n"
	p := writeTestFile(t, dir, "config_edge_test.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "TEST/domain/config(edge_cases)" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "TEST/domain/config(edge_cases)")
	}
	if sc.Version != 2 {
		t.Errorf("Version = %d, want 2", sc.Version)
	}
}

// TestParseSpecComment_CommentNotOnFirstLine verifies that the spec comment
// is found even when preceded by other lines (shebang, license, etc.).
func TestParseSpecComment_CommentNotOnFirstLine(t *testing.T) {
	dir := t.TempDir()
	content := "#!/usr/bin/env python3\n# License: MIT\n# spec: ROOT/scripts/deploy@v1\n"
	p := writeTestFile(t, dir, "deploy.py", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/scripts/deploy" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/scripts/deploy")
	}
	if sc.Version != 1 {
		t.Errorf("Version = %d, want 1", sc.Version)
	}
}

// TestParseSpecComment_SQLStyleComment verifies that a SQL-style "--" comment
// containing the spec reference is correctly parsed.
func TestParseSpecComment_SQLStyleComment(t *testing.T) {
	dir := t.TempDir()
	content := "-- spec: ROOT/database/migrations@v4\n"
	p := writeTestFile(t, dir, "migrations.sql", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/database/migrations" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/database/migrations")
	}
	if sc.Version != 4 {
		t.Errorf("Version = %d, want 4", sc.Version)
	}
}

// --- Edge Cases ---

// TestParseSpecComment_DeepInFile verifies that the spec comment is found
// even when buried after 100 lines of code.
func TestParseSpecComment_DeepInFile(t *testing.T) {
	dir := t.TempDir()

	// Build a file with 100 lines of code before the spec comment.
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "// some code line")
	}
	lines = append(lines, "// spec: ROOT/deep/node@v7")
	content := strings.Join(lines, "\n") + "\n"

	p := writeTestFile(t, dir, "deep.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/deep/node" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/deep/node")
	}
	if sc.Version != 7 {
		t.Errorf("Version = %d, want 7", sc.Version)
	}
}

// TestParseSpecComment_TrailingWhitespace verifies that trailing whitespace
// after the version number is correctly ignored.
func TestParseSpecComment_TrailingWhitespace(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/node@v3   \n"
	p := writeTestFile(t, dir, "trailing.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/node" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/node")
	}
	if sc.Version != 3 {
		t.Errorf("Version = %d, want 3", sc.Version)
	}
}

// TestParseSpecComment_ManySegments verifies that a logical name with many
// path segments is correctly parsed without truncation.
func TestParseSpecComment_ManySegments(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/a/b/c/d/e/f@v1\n"
	p := writeTestFile(t, dir, "deep_path.go", content)

	sc, err := ParseSpecComment(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sc.LogicalName != "ROOT/a/b/c/d/e/f" {
		t.Errorf("LogicalName = %q, want %q", sc.LogicalName, "ROOT/a/b/c/d/e/f")
	}
	if sc.Version != 1 {
		t.Errorf("Version = %d, want 1", sc.Version)
	}
}

// --- Failure Cases ---

// TestParseSpecComment_FileDoesNotExist verifies that calling ParseSpecComment
// with a non-existent path returns an error containing the file path.
func TestParseSpecComment_FileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "does_not_exist.go")

	_, err := ParseSpecComment(fakePath)
	if err == nil {
		t.Fatal("expected an error for non-existent file, got nil")
	}

	// The error message must contain the file path so the user knows
	// which file failed.
	if !strings.Contains(err.Error(), fakePath) {
		t.Errorf("error should contain file path %q, got: %s", fakePath, err.Error())
	}
}

// TestParseSpecComment_NoSpecComment verifies that a file with no "spec: "
// substring returns an error indicating no spec comment was found.
func TestParseSpecComment_NoSpecComment(t *testing.T) {
	dir := t.TempDir()
	content := "package main\n\nfunc main() {}\n"
	p := writeTestFile(t, dir, "main.go", content)

	_, err := ParseSpecComment(p)
	if err == nil {
		t.Fatal("expected an error for missing spec comment, got nil")
	}

	if !strings.Contains(err.Error(), "no spec comment found") {
		t.Errorf("error should contain %q, got: %s", "no spec comment found", err.Error())
	}
}

// TestParseSpecComment_MissingVersion verifies that a spec comment with "@v"
// but no version number after it returns a malformed error.
func TestParseSpecComment_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/node@v\n"
	p := writeTestFile(t, dir, "missing_ver.go", content)

	_, err := ParseSpecComment(p)
	if err == nil {
		t.Fatal("expected an error for missing version, got nil")
	}

	if !strings.Contains(err.Error(), "malformed spec comment") {
		t.Errorf("error should contain %q, got: %s", "malformed spec comment", err.Error())
	}
}

// TestParseSpecComment_MissingAtVSeparator verifies that a spec comment
// without the "@v" separator returns a malformed error.
func TestParseSpecComment_MissingAtVSeparator(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/node\n"
	p := writeTestFile(t, dir, "missing_atv.go", content)

	_, err := ParseSpecComment(p)
	if err == nil {
		t.Fatal("expected an error for missing @v separator, got nil")
	}

	if !strings.Contains(err.Error(), "malformed spec comment") {
		t.Errorf("error should contain %q, got: %s", "malformed spec comment", err.Error())
	}
}

// TestParseSpecComment_NonIntegerVersion verifies that a spec comment with
// a non-integer version string returns a malformed error.
func TestParseSpecComment_NonIntegerVersion(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: ROOT/node@vabc\n"
	p := writeTestFile(t, dir, "bad_ver.go", content)

	_, err := ParseSpecComment(p)
	if err == nil {
		t.Fatal("expected an error for non-integer version, got nil")
	}

	if !strings.Contains(err.Error(), "malformed spec comment") {
		t.Errorf("error should contain %q, got: %s", "malformed spec comment", err.Error())
	}
}

// TestParseSpecComment_EmptyLogicalName verifies that a spec comment with
// an empty logical name (just "@v<n>") returns a malformed error.
func TestParseSpecComment_EmptyLogicalName(t *testing.T) {
	dir := t.TempDir()
	content := "// spec: @v3\n"
	p := writeTestFile(t, dir, "empty_name.go", content)

	_, err := ParseSpecComment(p)
	if err == nil {
		t.Fatal("expected an error for empty logical name, got nil")
	}

	if !strings.Contains(err.Error(), "malformed spec comment") {
		t.Errorf("error should contain %q, got: %s", "malformed spec comment", err.Error())
	}
}
