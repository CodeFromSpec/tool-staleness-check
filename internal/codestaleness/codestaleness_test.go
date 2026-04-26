// code-from-spec: TEST/tech_design/internal/code_staleness@v13
package codestaleness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/discovery"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/frontmatter"
)

// testIntPtr is a helper to create *int values for Frontmatter.Version.
// Prefixed with "test" per test convention (ROOT/tech_design).
func testIntPtr(v int) *int {
	return &v
}

// testWriteFile creates a file in the given directory with the specified content.
// It uses t.Fatal on error so tests don't proceed with missing fixtures.
// Prefixed with "test" per test convention (ROOT/tech_design).
func testWriteFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file %s: %v", path, err)
	}
	return path
}

// --- Happy Path ---

// TestAllFilesUpToDate verifies that when a single implemented file
// has the correct spec comment and matching version, no results are returned.
func TestAllFilesUpToDate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the implemented file with correct spec comment and version.
	configPath := testWriteFile(t, tmpDir, "config.go",
		"// code-from-spec: ROOT/domain/config@v2\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(2),
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// TestMultipleFilesAllUpToDate verifies that when multiple implemented files
// all have correct spec comments and matching versions, no results are returned.
func TestMultipleFilesAllUpToDate(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := testWriteFile(t, tmpDir, "config.go",
		"// code-from-spec: ROOT/domain/config@v3\npackage config\n")
	utilPath := testWriteFile(t, tmpDir, "util.go",
		"// code-from-spec: ROOT/domain/config@v3\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(3),
			Implements: []string{configPath, utilPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// TestTestNodeCanonicalEquivalence verifies that LogicalNamesMatch treats
// TEST/domain/config and TEST/domain/config(default) as equivalent.
func TestTestNodeCanonicalEquivalence(t *testing.T) {
	tmpDir := t.TempDir()

	// The spec comment uses the (default) form, but the node's logical name
	// is the bare form. LogicalNamesMatch should treat them as equal.
	testPath := testWriteFile(t, tmpDir, "config_test.go",
		"// code-from-spec: TEST/domain/config(default)@v1\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:    testIntPtr(1),
			Implements: []string{testPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// TestNoImplements verifies that a node with no implements field
// returns an empty slice (step 3: nothing to check).
func TestNoImplements(t *testing.T) {
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/_node.md": {
			Version:    testIntPtr(5),
			Implements: nil,
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain",
		FilePath:    "code-from-spec/domain/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty results, got %v", results)
	}
}

// --- Blocking Steps (1-2) ---

// TestNodeNotInCache verifies that when the node's file path is not
// present in the cache at all, we get unreadable_frontmatter.
func TestNodeNotInCache(t *testing.T) {
	// Empty cache — node's file path has no entry.
	cache := map[string]*frontmatter.Frontmatter{}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// When frontmatter is not available, File should be empty.
	if results[0].Node != "ROOT/domain/config" {
		t.Errorf("expected Node=ROOT/domain/config, got %q", results[0].Node)
	}
	if results[0].File != "" {
		t.Errorf("expected File=\"\", got %q", results[0].File)
	}
	if results[0].Status != "unreadable_frontmatter" {
		t.Errorf("expected Status=unreadable_frontmatter, got %q", results[0].Status)
	}
}

// TestNodeNilInCache verifies that when the cache entry is nil
// (frontmatter parsing failed), we get unreadable_frontmatter.
func TestNodeNilInCache(t *testing.T) {
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": nil,
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "unreadable_frontmatter" {
		t.Errorf("expected Status=unreadable_frontmatter, got %q", results[0].Status)
	}
}

// TestVersionNil verifies that when the frontmatter has no version
// (Version is nil), we get no_version.
func TestVersionNil(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.go")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    nil,
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "no_version" {
		t.Errorf("expected Status=no_version, got %q", results[0].Status)
	}
}

// --- Per-file Statuses ---

// TestMissingFile verifies that when an implemented file does not exist,
// we get status "missing".
func TestMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Do not create the file — it should be reported as missing.
	nonexistentPath := filepath.Join(tmpDir, "nonexistent.go")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(2),
			Implements: []string{nonexistentPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Node != "ROOT/domain/config" {
		t.Errorf("expected Node=ROOT/domain/config, got %q", results[0].Node)
	}
	if results[0].File != nonexistentPath {
		t.Errorf("expected File=%q, got %q", nonexistentPath, results[0].File)
	}
	if results[0].Status != "missing" {
		t.Errorf("expected Status=missing, got %q", results[0].Status)
	}
}

// TestNoSpecComment verifies that when a file exists but contains
// no spec comment, we get status "no_spec_comment".
func TestNoSpecComment(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := testWriteFile(t, tmpDir, "config.go",
		"package config\n\nfunc Init() {}\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(2),
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "no_spec_comment" {
		t.Errorf("expected Status=no_spec_comment, got %q", results[0].Status)
	}
}

// TestMalformedSpecComment verifies that when a file has a spec comment
// that cannot be parsed (e.g., non-numeric version), we get "malformed_spec_comment".
func TestMalformedSpecComment(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := testWriteFile(t, tmpDir, "config.go",
		"// code-from-spec: ROOT/domain/config@vabc\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(2),
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "malformed_spec_comment" {
		t.Errorf("expected Status=malformed_spec_comment, got %q", results[0].Status)
	}
}

// TestWrongNode verifies that when a file's spec comment references
// a different node than expected, we get "wrong_node".
func TestWrongNode(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := testWriteFile(t, tmpDir, "config.go",
		"// code-from-spec: ROOT/domain/other@v2\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(2),
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "wrong_node" {
		t.Errorf("expected Status=wrong_node, got %q", results[0].Status)
	}
}

// TestStale verifies that when a file's spec comment version does not
// match the node's current version, we get "stale".
func TestStale(t *testing.T) {
	tmpDir := t.TempDir()

	// Node is at version 3, but spec comment says v2.
	configPath := testWriteFile(t, tmpDir, "config.go",
		"// code-from-spec: ROOT/domain/config@v2\npackage config\n")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(3),
			Implements: []string{configPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "stale" {
		t.Errorf("expected Status=stale, got %q", results[0].Status)
	}
}

// --- Multiple Files ---

// TestMixedResultsAcrossFiles verifies that when a node implements multiple
// files with different conditions, results are returned only for problematic
// files, in the correct order.
func TestMixedResultsAcrossFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// a.go is up to date (v3 matches).
	aPath := testWriteFile(t, tmpDir, "a.go",
		"// code-from-spec: ROOT/domain/config@v3\npackage config\n")

	// b.go is stale (v2 != v3).
	bPath := testWriteFile(t, tmpDir, "b.go",
		"// code-from-spec: ROOT/domain/config@v2\npackage config\n")

	// c.go does not exist — missing.
	cPath := filepath.Join(tmpDir, "c.go")

	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:    testIntPtr(3),
			Implements: []string{aPath, bPath, cPath},
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckCodeStaleness(node, cache)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}

	// First result: b.go is stale.
	if results[0].File != bPath {
		t.Errorf("results[0]: expected File=%q, got %q", bPath, results[0].File)
	}
	if results[0].Status != "stale" {
		t.Errorf("results[0]: expected Status=stale, got %q", results[0].Status)
	}

	// Second result: c.go is missing.
	if results[1].File != cPath {
		t.Errorf("results[1]: expected File=%q, got %q", cPath, results[1].File)
	}
	if results[1].Status != "missing" {
		t.Errorf("results[1]: expected Status=missing, got %q", results[1].Status)
	}
}
