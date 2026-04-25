// spec: TEST/tech_design/discovery@v3
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper: createFile creates a file at the given path relative to dir,
// creating intermediate directories as needed. The file content is
// minimal frontmatter since discovery only checks file existence and names.
func createFile(t *testing.T, dir, relPath string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating directories for %s: %v", relPath, err)
	}
	if err := os.WriteFile(full, []byte("---\nversion: 1\n---\n"), 0o644); err != nil {
		t.Fatalf("creating file %s: %v", relPath, err)
	}
}

// helper: mkdirAll creates a directory (and parents) inside dir.
// Fatals the test on error so callers never leave an error unchecked.
func mkdirAll(t *testing.T, dir, relPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, relPath), 0o755); err != nil {
		t.Fatalf("creating directory %s: %v", relPath, err)
	}
}

// helper: chdirTo changes the working directory to dir and registers
// a cleanup function that restores the original directory. This avoids
// interference between tests.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to directory %s: %v", dir, err)
	}
	t.Cleanup(func() {
		// Restore the original working directory. If this fails the test
		// environment is compromised, so we log the error. We cannot call
		// t.Fatalf inside Cleanup after the test has already finished, but
		// t.Logf is safe and ensures the failure is visible.
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restoring working directory to %s: %v", orig, err)
		}
	})
}

// --- Happy Path ---

// TestDiscoverNodes_SpecNodesAtAllLevels verifies that _node.md files at
// root, intermediate, and leaf levels are all discovered with the correct
// ROOT/ logical names.
func TestDiscoverNodes_SpecNodesAtAllLevels(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/config/_node.md")
	// code-from-spec/external/ must exist to avoid error ambiguity; empty is fine.
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	specNodes, _, _, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect exactly three spec nodes.
	if len(specNodes) != 3 {
		t.Fatalf("expected 3 spec nodes, got %d", len(specNodes))
	}

	// Verify logical names and file paths.
	expected := []struct {
		name string
		path string
	}{
		{"ROOT", "code-from-spec/spec/_node.md"},
		{"ROOT/domain", "code-from-spec/spec/domain/_node.md"},
		{"ROOT/domain/config", "code-from-spec/spec/domain/config/_node.md"},
	}
	for i, want := range expected {
		if specNodes[i].LogicalName != want.name {
			t.Errorf("specNodes[%d].LogicalName = %q, want %q", i, specNodes[i].LogicalName, want.name)
		}
		if specNodes[i].FilePath != want.path {
			t.Errorf("specNodes[%d].FilePath = %q, want %q", i, specNodes[i].FilePath, want.path)
		}
	}
}

// TestDiscoverNodes_TestNodes verifies that default.test.md and named
// *.test.md files produce correct TEST/ logical names.
func TestDiscoverNodes_TestNodes(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/domain/config/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/config/default.test.md")
	createFile(t, dir, "code-from-spec/spec/domain/config/edge_cases.test.md")
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	_, testNodes, _, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(testNodes) != 2 {
		t.Fatalf("expected 2 test nodes, got %d", len(testNodes))
	}

	// Sorted alphabetically by LogicalName:
	// TEST/domain/config < TEST/domain/config(edge_cases)
	expected := []struct {
		name string
		path string
	}{
		{"TEST/domain/config", "code-from-spec/spec/domain/config/default.test.md"},
		{"TEST/domain/config(edge_cases)", "code-from-spec/spec/domain/config/edge_cases.test.md"},
	}
	for i, want := range expected {
		if testNodes[i].LogicalName != want.name {
			t.Errorf("testNodes[%d].LogicalName = %q, want %q", i, testNodes[i].LogicalName, want.name)
		}
		if testNodes[i].FilePath != want.path {
			t.Errorf("testNodes[%d].FilePath = %q, want %q", i, testNodes[i].FilePath, want.path)
		}
	}
}

// TestDiscoverNodes_ExternalDependencies verifies that _external.md files
// under code-from-spec/external/ produce EXTERNAL/ logical names.
func TestDiscoverNodes_ExternalDependencies(t *testing.T) {
	dir := t.TempDir()
	// code-from-spec/spec/ must exist for DiscoverNodes to succeed.
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/external/database/_external.md")
	createFile(t, dir, "code-from-spec/external/celcoin-api/_external.md")

	chdirTo(t, dir)

	_, _, externalDeps, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(externalDeps) != 2 {
		t.Fatalf("expected 2 external deps, got %d", len(externalDeps))
	}

	// Sorted alphabetically: EXTERNAL/celcoin-api < EXTERNAL/database
	expected := []struct {
		name string
		path string
	}{
		{"EXTERNAL/celcoin-api", "code-from-spec/external/celcoin-api/_external.md"},
		{"EXTERNAL/database", "code-from-spec/external/database/_external.md"},
	}
	for i, want := range expected {
		if externalDeps[i].LogicalName != want.name {
			t.Errorf("externalDeps[%d].LogicalName = %q, want %q", i, externalDeps[i].LogicalName, want.name)
		}
		if externalDeps[i].FilePath != want.path {
			t.Errorf("externalDeps[%d].FilePath = %q, want %q", i, externalDeps[i].FilePath, want.path)
		}
	}
}

// TestDiscoverNodes_SortedAlphabetically verifies all three lists are
// sorted by LogicalName regardless of filesystem order.
func TestDiscoverNodes_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()

	// Create spec nodes in an order where filesystem order might differ
	// from alphabetical order (e.g., "zebra" before "alpha").
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/spec/zebra/_node.md")
	createFile(t, dir, "code-from-spec/spec/alpha/_node.md")
	createFile(t, dir, "code-from-spec/spec/middle/_node.md")

	// Test nodes: create in reverse alphabetical order.
	createFile(t, dir, "code-from-spec/spec/alpha/default.test.md")
	createFile(t, dir, "code-from-spec/spec/zebra/default.test.md")

	// External deps: create in reverse alphabetical order.
	createFile(t, dir, "code-from-spec/external/zservice/_external.md")
	createFile(t, dir, "code-from-spec/external/aservice/_external.md")

	chdirTo(t, dir)

	specNodes, testNodes, externalDeps, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify spec nodes are sorted.
	expectedSpec := []string{"ROOT", "ROOT/alpha", "ROOT/middle", "ROOT/zebra"}
	if len(specNodes) != len(expectedSpec) {
		t.Fatalf("expected %d spec nodes, got %d", len(expectedSpec), len(specNodes))
	}
	for i, want := range expectedSpec {
		if specNodes[i].LogicalName != want {
			t.Errorf("specNodes[%d].LogicalName = %q, want %q", i, specNodes[i].LogicalName, want)
		}
	}

	// Verify test nodes are sorted.
	expectedTest := []string{"TEST/alpha", "TEST/zebra"}
	if len(testNodes) != len(expectedTest) {
		t.Fatalf("expected %d test nodes, got %d", len(expectedTest), len(testNodes))
	}
	for i, want := range expectedTest {
		if testNodes[i].LogicalName != want {
			t.Errorf("testNodes[%d].LogicalName = %q, want %q", i, testNodes[i].LogicalName, want)
		}
	}

	// Verify external deps are sorted.
	expectedExt := []string{"EXTERNAL/aservice", "EXTERNAL/zservice"}
	if len(externalDeps) != len(expectedExt) {
		t.Fatalf("expected %d external deps, got %d", len(expectedExt), len(externalDeps))
	}
	for i, want := range expectedExt {
		if externalDeps[i].LogicalName != want {
			t.Errorf("externalDeps[%d].LogicalName = %q, want %q", i, externalDeps[i].LogicalName, want)
		}
	}
}

// --- Edge Cases ---

// TestDiscoverNodes_EmptySpecDirectory verifies that an empty
// code-from-spec/spec/ directory produces no spec or test nodes and no error.
func TestDiscoverNodes_EmptySpecDirectory(t *testing.T) {
	dir := t.TempDir()
	mkdirAll(t, dir, "code-from-spec/spec")
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	specNodes, testNodes, _, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specNodes) != 0 {
		t.Errorf("expected 0 spec nodes, got %d", len(specNodes))
	}
	if len(testNodes) != 0 {
		t.Errorf("expected 0 test nodes, got %d", len(testNodes))
	}
}

// TestDiscoverNodes_EmptyExternalDirectory verifies that a
// code-from-spec/external/ directory with no subdirectories produces no
// external deps and no error.
func TestDiscoverNodes_EmptyExternalDirectory(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/_node.md")
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	_, _, externalDeps, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(externalDeps) != 0 {
		t.Errorf("expected 0 external deps, got %d", len(externalDeps))
	}
}

// TestDiscoverNodes_NonNodeFilesIgnored verifies that files in
// code-from-spec/spec/ that are not _node.md or *.test.md are excluded
// from all lists.
func TestDiscoverNodes_NonNodeFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/spec/README.md")
	createFile(t, dir, "code-from-spec/spec/notes.txt")
	createFile(t, dir, "code-from-spec/spec/domain/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/README.md")
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	specNodes, testNodes, _, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only _node.md files should appear as spec nodes.
	if len(specNodes) != 2 {
		t.Fatalf("expected 2 spec nodes, got %d", len(specNodes))
	}
	if specNodes[0].LogicalName != "ROOT" {
		t.Errorf("specNodes[0].LogicalName = %q, want %q", specNodes[0].LogicalName, "ROOT")
	}
	if specNodes[1].LogicalName != "ROOT/domain" {
		t.Errorf("specNodes[1].LogicalName = %q, want %q", specNodes[1].LogicalName, "ROOT/domain")
	}

	// No test nodes should be discovered (README.md and notes.txt are not test files).
	if len(testNodes) != 0 {
		t.Errorf("expected 0 test nodes, got %d", len(testNodes))
	}
}

// TestDiscoverNodes_ExternalDirectoryWithExtraFiles verifies that extra
// files alongside _external.md in an external dependency folder are ignored.
func TestDiscoverNodes_ExternalDirectoryWithExtraFiles(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/external/database/_external.md")
	createFile(t, dir, "code-from-spec/external/database/schema.sql")
	createFile(t, dir, "code-from-spec/external/database/README.md")

	chdirTo(t, dir)

	_, _, externalDeps, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only one entry for EXTERNAL/database, extra files ignored.
	if len(externalDeps) != 1 {
		t.Fatalf("expected 1 external dep, got %d", len(externalDeps))
	}
	if externalDeps[0].LogicalName != "EXTERNAL/database" {
		t.Errorf("externalDeps[0].LogicalName = %q, want %q", externalDeps[0].LogicalName, "EXTERNAL/database")
	}
	if externalDeps[0].FilePath != "code-from-spec/external/database/_external.md" {
		t.Errorf("externalDeps[0].FilePath = %q, want %q", externalDeps[0].FilePath, "code-from-spec/external/database/_external.md")
	}
}

// --- Failure Cases ---

// TestDiscoverNodes_SpecDirectoryNotExist verifies that an error is
// returned when the code-from-spec/spec/ directory does not exist, and
// that the error message is descriptive enough for the user to understand
// the problem.
func TestDiscoverNodes_SpecDirectoryNotExist(t *testing.T) {
	dir := t.TempDir()
	// Do not create code-from-spec/spec/. Only create code-from-spec/external/
	// so it is not the cause.
	mkdirAll(t, dir, "code-from-spec/external")

	chdirTo(t, dir)

	_, _, _, err := DiscoverNodes()
	if err == nil {
		t.Fatal("expected an error when code-from-spec/spec/ does not exist, got nil")
	}

	// The error message must be descriptive so the caller can print it
	// directly. Per the discovery spec, it should contain something like
	// "code-from-spec/spec/ directory not found".
	msg := err.Error()
	if !strings.Contains(msg, "code-from-spec/spec/ directory not found") {
		t.Errorf("error message should contain %q, got: %s", "code-from-spec/spec/ directory not found", msg)
	}
}

// TestDiscoverNodes_ExternalDirectoryNotExist verifies that when
// code-from-spec/external/ does not exist, spec and test discovery still
// succeeds and externalDeps is empty with no error.
func TestDiscoverNodes_ExternalDirectoryNotExist(t *testing.T) {
	dir := t.TempDir()
	createFile(t, dir, "code-from-spec/spec/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/_node.md")
	createFile(t, dir, "code-from-spec/spec/domain/default.test.md")
	// Do not create code-from-spec/external/.

	chdirTo(t, dir)

	specNodes, testNodes, externalDeps, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Spec and test discovery should still work.
	if len(specNodes) != 2 {
		t.Errorf("expected 2 spec nodes, got %d", len(specNodes))
	}
	if len(testNodes) != 1 {
		t.Errorf("expected 1 test node, got %d", len(testNodes))
	}

	// externalDeps should be empty since code-from-spec/external/ does not exist.
	if len(externalDeps) != 0 {
		t.Errorf("expected 0 external deps, got %d", len(externalDeps))
	}
}
