// spec: TEST/tech_design/code_staleness@v1
package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeCodeFile is a test helper that creates a file at the given path with
// the given content. Used to set up generated code files for code staleness
// tests. It uses t.Fatalf on failure so every error is checked.
func writeCodeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
}

// ---------------------------------------------------------------------------
// Happy Path
// ---------------------------------------------------------------------------

// TestAllFilesUpToDate verifies that when all implemented files have a spec
// comment matching the node's logical name and version, CheckCodeStaleness
// returns an empty slice.
func TestAllFilesUpToDate(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// Create a file with the correct spec comment matching version 2.
	writeCodeFile(t, configPath, "// spec: ROOT/domain/config@v2\npackage config\n")

	// Build the cache with a matching version and implements pointing to
	// the temp file.
	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(2),
			Implements: []string{configPath},
		},
	}

	// The discovered node for this spec.
	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: empty slice — all files are up to date.
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d: %+v", len(results), results)
	}
}

// TestMultipleFilesAllUpToDate verifies that when a node implements multiple
// files and all have correct spec comments, the result is an empty slice.
func TestMultipleFilesAllUpToDate(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")
	utilPath := filepath.Join(tmpdir, "util.go")

	// Both files have spec comments matching version 3.
	writeCodeFile(t, configPath, "// spec: ROOT/domain/config@v3\npackage config\n")
	writeCodeFile(t, utilPath, "// spec: ROOT/domain/config@v3\npackage config\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(3),
			Implements: []string{configPath, utilPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: empty slice — both files match.
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d: %+v", len(results), results)
	}
}

// TestCanonicalEquivalence verifies that LogicalNamesMatch treats
// TEST/domain/config and TEST/domain/config(default) as equal. The node's
// logical name is TEST/domain/config but the file's spec comment uses the
// explicit form TEST/domain/config(default). The result should still be
// empty (up to date).
func TestCanonicalEquivalence(t *testing.T) {
	tmpdir := t.TempDir()
	testPath := filepath.Join(tmpdir, "config_test.go")

	// The file uses the explicit (default) form in its spec comment.
	writeCodeFile(t, testPath, "// spec: TEST/domain/config(default)@v1\npackage config\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/default.test.md": {
			Version:    intPtr(1),
			Implements: []string{testPath},
		},
	}

	// The discovered node uses the short form without (default).
	node := DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "spec/domain/config/default.test.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: empty slice — TEST/domain/config and TEST/domain/config(default)
	// are canonically equivalent.
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d: %+v", len(results), results)
	}
}

// TestNoImplements verifies that a node with no implements field produces
// an empty result slice (nothing to check).
func TestNoImplements(t *testing.T) {
	cache := map[string]*Frontmatter{
		"spec/domain/_node.md": {
			Version:    intPtr(5),
			Implements: nil,
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain",
		FilePath:    "spec/domain/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: empty slice — no implements means nothing to check.
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d: %+v", len(results), results)
	}
}

// ---------------------------------------------------------------------------
// Blocking Steps (1-2)
// ---------------------------------------------------------------------------

// TestCodeStaleness_NodeNotInCache verifies that when the node's file path
// has no entry in the cache at all, CheckCodeStaleness returns
// unreadable_frontmatter.
func TestCodeStaleness_NodeNotInCache(t *testing.T) {
	// Empty cache — the node's file path is not present.
	cache := map[string]*Frontmatter{}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with unreadable_frontmatter.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Node != "ROOT/domain/config" {
		t.Fatalf("expected Node=%q, got %q", "ROOT/domain/config", results[0].Node)
	}
	if results[0].File != "" {
		t.Fatalf("expected File=%q, got %q", "", results[0].File)
	}
	if results[0].Status != "unreadable_frontmatter" {
		t.Fatalf("expected Status=%q, got %q", "unreadable_frontmatter", results[0].Status)
	}
}

// TestNodeNilInCache verifies that when the node's cache entry is nil
// (frontmatter parsing failed), CheckCodeStaleness returns
// unreadable_frontmatter.
func TestNodeNilInCache(t *testing.T) {
	// Cache has the key but with a nil value — parsing failed.
	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": nil,
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with unreadable_frontmatter.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "unreadable_frontmatter" {
		t.Fatalf("expected Status=%q, got %q", "unreadable_frontmatter", results[0].Status)
	}
}

// TestVersionNil verifies that when frontmatter.Version is nil (version field
// absent from YAML), CheckCodeStaleness returns no_version.
func TestVersionNil(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// The file exists but Version is nil — we should not even get to the
	// file check. Still, the implements list must point to a real path
	// so the cache entry is realistic.
	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    nil,
			Implements: []string{configPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with no_version.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "no_version" {
		t.Fatalf("expected Status=%q, got %q", "no_version", results[0].Status)
	}
}

// ---------------------------------------------------------------------------
// Per-file Statuses
// ---------------------------------------------------------------------------

// TestMissingFile verifies that when an implemented file does not exist on
// disk, CheckCodeStaleness returns status "missing".
func TestMissingFile(t *testing.T) {
	tmpdir := t.TempDir()
	// Point to a file that was never created.
	nonexistentPath := filepath.Join(tmpdir, "nonexistent.go")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(2),
			Implements: []string{nonexistentPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with status "missing" and the file path set.
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Node != "ROOT/domain/config" {
		t.Fatalf("expected Node=%q, got %q", "ROOT/domain/config", results[0].Node)
	}
	if results[0].File != nonexistentPath {
		t.Fatalf("expected File=%q, got %q", nonexistentPath, results[0].File)
	}
	if results[0].Status != "missing" {
		t.Fatalf("expected Status=%q, got %q", "missing", results[0].Status)
	}
}

// TestNoSpecComment verifies that when a file exists but contains no "spec:"
// content, CheckCodeStaleness returns status "no_spec_comment".
func TestNoSpecComment(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// Create a file without any spec comment.
	writeCodeFile(t, configPath, "package config\n\nfunc Init() {}\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(2),
			Implements: []string{configPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with status "no_spec_comment".
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "no_spec_comment" {
		t.Fatalf("expected Status=%q, got %q", "no_spec_comment", results[0].Status)
	}
}

// TestMalformedSpecComment verifies that when a file contains a spec comment
// with a non-integer version (e.g., @vabc), CheckCodeStaleness returns
// status "malformed_spec_comment".
func TestMalformedSpecComment(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// Create a file with a malformed spec comment — version is "abc", not
	// a valid integer.
	writeCodeFile(t, configPath, "// spec: ROOT/domain/config@vabc\npackage config\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(2),
			Implements: []string{configPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with status "malformed_spec_comment".
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "malformed_spec_comment" {
		t.Fatalf("expected Status=%q, got %q", "malformed_spec_comment", results[0].Status)
	}
}

// TestWrongNode verifies that when a file's spec comment references a
// different logical name than the node's, CheckCodeStaleness returns
// status "wrong_node".
func TestWrongNode(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// Create a file referencing ROOT/domain/other instead of
	// ROOT/domain/config.
	writeCodeFile(t, configPath, "// spec: ROOT/domain/other@v2\npackage config\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(2),
			Implements: []string{configPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with status "wrong_node".
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "wrong_node" {
		t.Fatalf("expected Status=%q, got %q", "wrong_node", results[0].Status)
	}
}

// TestStale verifies that when a file's spec comment version does not match
// the node's version, CheckCodeStaleness returns status "stale".
func TestStale(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.go")

	// Create a file with version 2, but the node's version is 3.
	writeCodeFile(t, configPath, "// spec: ROOT/domain/config@v2\npackage config\n")

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(3),
			Implements: []string{configPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: single result with status "stale".
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "stale" {
		t.Fatalf("expected Status=%q, got %q", "stale", results[0].Status)
	}
}

// ---------------------------------------------------------------------------
// Multiple Files
// ---------------------------------------------------------------------------

// TestMixedResultsAcrossFiles verifies that when a node implements multiple
// files with mixed statuses, CheckCodeStaleness returns one result per
// problematic file and omits up-to-date files.
func TestMixedResultsAcrossFiles(t *testing.T) {
	tmpdir := t.TempDir()
	aPath := filepath.Join(tmpdir, "a.go")
	bPath := filepath.Join(tmpdir, "b.go")
	cPath := filepath.Join(tmpdir, "c.go") // deliberately not created

	// a.go is up to date (version 3 matches).
	writeCodeFile(t, aPath, "// spec: ROOT/domain/config@v3\npackage config\n")

	// b.go is stale (version 2, but node version is 3).
	writeCodeFile(t, bPath, "// spec: ROOT/domain/config@v2\npackage config\n")

	// c.go is not created — should be reported as "missing".

	cache := map[string]*Frontmatter{
		"spec/domain/config/_node.md": {
			Version:    intPtr(3),
			Implements: []string{aPath, bPath, cPath},
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)

	// Expect: two results — b.go stale and c.go missing. a.go is omitted
	// because it is up to date.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}

	// First result should be for b.go (stale) since it comes before c.go
	// in the implements list order.
	if results[0].File != bPath {
		t.Fatalf("expected results[0].File=%q, got %q", bPath, results[0].File)
	}
	if results[0].Status != "stale" {
		t.Fatalf("expected results[0].Status=%q, got %q", "stale", results[0].Status)
	}

	// Second result should be for c.go (missing).
	if results[1].File != cPath {
		t.Fatalf("expected results[1].File=%q, got %q", cPath, results[1].File)
	}
	if results[1].Status != "missing" {
		t.Fatalf("expected results[1].Status=%q, got %q", "missing", results[1].Status)
	}
}
