// code-from-spec: TEST/tech_design/internal/discovery@v8
package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: createFile creates a file (and its parent directories) inside baseDir.
// The filePath is slash-separated and relative to baseDir.
func createFile(t *testing.T, baseDir, filePath string) {
	t.Helper()
	full := filepath.Join(baseDir, filepath.FromSlash(filePath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("failed to create directories for %s: %v", filePath, err)
	}
	if err := os.WriteFile(full, []byte("---\nversion: 1\n---\n# placeholder\n"), 0o644); err != nil {
		t.Fatalf("failed to create file %s: %v", filePath, err)
	}
}

// helper: chdirTo changes the working directory to dir and returns a cleanup
// function that restores the original working directory.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	})
}

// TestDiscoverNodes_SpecNodesAtAllLevels verifies that _node.md files at
// root, intermediate, and leaf levels are all discovered as spec nodes
// with correct logical names and file paths.
func TestDiscoverNodes_SpecNodesAtAllLevels(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec nodes at three levels.
	createFile(t, tmpDir, "code-from-spec/_node.md")
	createFile(t, tmpDir, "code-from-spec/domain/_node.md")
	createFile(t, tmpDir, "code-from-spec/domain/config/_node.md")

	chdirTo(t, tmpDir)

	specNodes, testNodes, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect exactly 3 spec nodes.
	if len(specNodes) != 3 {
		t.Fatalf("expected 3 spec nodes, got %d", len(specNodes))
	}

	// Verify each spec node's logical name and file path.
	expectedSpec := []DiscoveredNode{
		{LogicalName: "ROOT", FilePath: "code-from-spec/_node.md"},
		{LogicalName: "ROOT/domain", FilePath: "code-from-spec/domain/_node.md"},
		{LogicalName: "ROOT/domain/config", FilePath: "code-from-spec/domain/config/_node.md"},
	}
	for i, want := range expectedSpec {
		got := specNodes[i]
		if got.LogicalName != want.LogicalName || got.FilePath != want.FilePath {
			t.Errorf("specNodes[%d] = {%q, %q}, want {%q, %q}",
				i, got.LogicalName, got.FilePath, want.LogicalName, want.FilePath)
		}
	}

	// No test nodes expected.
	if len(testNodes) != 0 {
		t.Errorf("expected 0 test nodes, got %d", len(testNodes))
	}
}

// TestDiscoverNodes_TestNodes verifies that *.test.md files are discovered
// as test nodes with correct logical names (including named variants).
func TestDiscoverNodes_TestNodes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec node and two test nodes in the same directory.
	createFile(t, tmpDir, "code-from-spec/domain/config/_node.md")
	createFile(t, tmpDir, "code-from-spec/domain/config/default.test.md")
	createFile(t, tmpDir, "code-from-spec/domain/config/edge_cases.test.md")

	chdirTo(t, tmpDir)

	_, testNodes, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect exactly 2 test nodes, sorted by LogicalName.
	if len(testNodes) != 2 {
		t.Fatalf("expected 2 test nodes, got %d", len(testNodes))
	}

	expectedTest := []DiscoveredNode{
		{LogicalName: "TEST/domain/config", FilePath: "code-from-spec/domain/config/default.test.md"},
		{LogicalName: "TEST/domain/config(edge_cases)", FilePath: "code-from-spec/domain/config/edge_cases.test.md"},
	}
	for i, want := range expectedTest {
		got := testNodes[i]
		if got.LogicalName != want.LogicalName || got.FilePath != want.FilePath {
			t.Errorf("testNodes[%d] = {%q, %q}, want {%q, %q}",
				i, got.LogicalName, got.FilePath, want.LogicalName, want.FilePath)
		}
	}
}

// TestDiscoverNodes_TestNodesAlongsideIntermediateNodes verifies that test
// nodes placed alongside intermediate (non-leaf) spec nodes are discovered.
func TestDiscoverNodes_TestNodesAlongsideIntermediateNodes(t *testing.T) {
	tmpDir := t.TempDir()

	// domain/ is intermediate (has child config/), but also has a test node.
	createFile(t, tmpDir, "code-from-spec/domain/_node.md")
	createFile(t, tmpDir, "code-from-spec/domain/default.test.md")
	createFile(t, tmpDir, "code-from-spec/domain/config/_node.md")

	chdirTo(t, tmpDir)

	_, testNodes, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify test node for domain/ is present.
	found := false
	for _, tn := range testNodes {
		if tn.LogicalName == "TEST/domain" && tn.FilePath == "code-from-spec/domain/default.test.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test node TEST/domain -> code-from-spec/domain/default.test.md, not found in %v", testNodes)
	}
}

// TestDiscoverNodes_SortedAlphabetically verifies that both specNodes and
// testNodes are sorted alphabetically by LogicalName, regardless of
// filesystem walk order.
func TestDiscoverNodes_SortedAlphabetically(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nodes in an order where filesystem walk order might differ
	// from alphabetical order of logical names. Directories starting with
	// 'z' come after 'a' alphabetically but we create them first.
	createFile(t, tmpDir, "code-from-spec/_node.md")
	createFile(t, tmpDir, "code-from-spec/zebra/_node.md")
	createFile(t, tmpDir, "code-from-spec/alpha/_node.md")
	createFile(t, tmpDir, "code-from-spec/middle/_node.md")
	createFile(t, tmpDir, "code-from-spec/zebra/default.test.md")
	createFile(t, tmpDir, "code-from-spec/alpha/default.test.md")

	chdirTo(t, tmpDir)

	specNodes, testNodes, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify spec nodes are sorted by LogicalName.
	for i := 1; i < len(specNodes); i++ {
		if specNodes[i].LogicalName < specNodes[i-1].LogicalName {
			t.Errorf("specNodes not sorted: %q comes after %q",
				specNodes[i].LogicalName, specNodes[i-1].LogicalName)
		}
	}

	// Verify test nodes are sorted by LogicalName.
	for i := 1; i < len(testNodes); i++ {
		if testNodes[i].LogicalName < testNodes[i-1].LogicalName {
			t.Errorf("testNodes not sorted: %q comes after %q",
				testNodes[i].LogicalName, testNodes[i-1].LogicalName)
		}
	}

	// Sanity check: alpha < middle < zebra in spec nodes (after ROOT).
	if len(specNodes) != 4 {
		t.Fatalf("expected 4 spec nodes, got %d", len(specNodes))
	}
	expectedOrder := []string{"ROOT", "ROOT/alpha", "ROOT/middle", "ROOT/zebra"}
	for i, want := range expectedOrder {
		if specNodes[i].LogicalName != want {
			t.Errorf("specNodes[%d].LogicalName = %q, want %q", i, specNodes[i].LogicalName, want)
		}
	}

	// Test nodes: TEST/alpha before TEST/zebra.
	if len(testNodes) != 2 {
		t.Fatalf("expected 2 test nodes, got %d", len(testNodes))
	}
	if testNodes[0].LogicalName != "TEST/alpha" {
		t.Errorf("testNodes[0].LogicalName = %q, want %q", testNodes[0].LogicalName, "TEST/alpha")
	}
	if testNodes[1].LogicalName != "TEST/zebra" {
		t.Errorf("testNodes[1].LogicalName = %q, want %q", testNodes[1].LogicalName, "TEST/zebra")
	}
}

// TestDiscoverNodes_EmptyCodeFromSpecDirectory verifies that an error is
// returned when code-from-spec/ exists but contains no _node.md files.
func TestDiscoverNodes_EmptyCodeFromSpecDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the directory but no _node.md files inside it.
	if err := os.MkdirAll(filepath.Join(tmpDir, "code-from-spec"), 0o755); err != nil {
		t.Fatalf("failed to create code-from-spec/: %v", err)
	}

	chdirTo(t, tmpDir)

	_, _, err := DiscoverNodes()
	if err == nil {
		t.Fatal("expected an error for empty code-from-spec/ directory, got nil")
	}
}

// TestDiscoverNodes_NonNodeFilesIgnored verifies that files that are neither
// _node.md nor *.test.md are not included in any result list.
func TestDiscoverNodes_NonNodeFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()

	// Create one valid spec node plus some non-node files.
	createFile(t, tmpDir, "code-from-spec/_node.md")
	createFile(t, tmpDir, "code-from-spec/README.md")
	createFile(t, tmpDir, "code-from-spec/notes.txt")
	createFile(t, tmpDir, "code-from-spec/domain/README.md")

	chdirTo(t, tmpDir)

	specNodes, testNodes, err := DiscoverNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the _node.md should appear; README.md and notes.txt must be absent.
	if len(specNodes) != 1 {
		t.Errorf("expected 1 spec node, got %d: %v", len(specNodes), specNodes)
	}
	if specNodes[0].LogicalName != "ROOT" {
		t.Errorf("expected spec node ROOT, got %q", specNodes[0].LogicalName)
	}

	if len(testNodes) != 0 {
		t.Errorf("expected 0 test nodes, got %d: %v", len(testNodes), testNodes)
	}
}

// TestDiscoverNodes_DirectoryDoesNotExist verifies that an error is returned
// when code-from-spec/ does not exist at all.
func TestDiscoverNodes_DirectoryDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Do not create code-from-spec/ at all.
	chdirTo(t, tmpDir)

	_, _, err := DiscoverNodes()
	if err == nil {
		t.Fatal("expected an error when code-from-spec/ does not exist, got nil")
	}
}
