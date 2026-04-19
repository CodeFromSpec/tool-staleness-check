// spec: TEST/tech_design/main@v1
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// binaryPath holds the absolute path to the compiled staleness-check binary.
// It is built once in TestMain and reused by all tests.
var binaryPath string

// TestMain compiles the staleness-check binary once and stores its path in
// binaryPath for all integration tests to use. The binary is placed in a
// temporary directory that is cleaned up after all tests finish.
func TestMain(m *testing.M) {
	// Create a temporary directory for the compiled binary.
	tmpDir, err := os.MkdirTemp("", "staleness-check-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir for binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build the binary into the temp directory. The "." source argument means
	// the current package (cmd/staleness-check/).
	binaryPath = filepath.Join(tmpDir, "staleness-check.exe")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build staleness-check binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// writeNodeFile creates a node file at the given path inside the temp directory.
// It writes valid YAML frontmatter between --- delimiters and a title line.
// The frontmatter map holds YAML key/value pairs. The title is the logical name
// written as "# <title>".
func writeNodeFile(t *testing.T, dir string, relPath string, frontmatter map[string]interface{}, title string) {
	t.Helper()

	fullPath := filepath.Join(dir, relPath)

	// Ensure parent directory exists.
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", parentDir, err)
	}

	// Marshal the frontmatter map to YAML.
	yamlBytes, err := yaml.Marshal(frontmatter)
	if err != nil {
		t.Fatalf("failed to marshal frontmatter for %s: %v", relPath, err)
	}

	// Build the file content: --- + YAML + --- + blank line + # title
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(yamlBytes)
	content.WriteString("---\n\n")
	content.WriteString("# " + title + "\n")

	if err := os.WriteFile(fullPath, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to write node file %s: %v", relPath, err)
	}
}

// writeGeneratedFile creates a generated Go file at the given path inside the
// temp directory. The file contains a spec comment line and a package declaration.
func writeGeneratedFile(t *testing.T, dir string, relPath string, specComment string) {
	t.Helper()

	fullPath := filepath.Join(dir, relPath)

	// Ensure parent directory exists.
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", parentDir, err)
	}

	content := specComment + "\npackage main\n"
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write generated file %s: %v", relPath, err)
	}
}

// runBinary executes the staleness-check binary with the given arguments,
// using workDir as the working directory. Returns stdout, stderr, and the
// exit code. The exit code is 0 if the command succeeds, or the ExitError
// code on failure.
func runBinary(t *testing.T, workDir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		// Extract exit code from ExitError.
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running binary: %v", err)
		}
	}

	return stdout, stderr, exitCode
}

// toolOutput is the structure for parsing the YAML output from the binary.
// It mirrors the output struct in main.go with exported fields and YAML tags.
type toolOutput struct {
	SpecStaleness []specStalenessOutputEntry `yaml:"spec_staleness"`
	TestStaleness []specStalenessOutputEntry `yaml:"test_staleness"`
	CodeStaleness []codeStalenessOutputEntry `yaml:"code_staleness"`
}

// specStalenessOutputEntry mirrors specStalenessEntry for YAML parsing.
type specStalenessOutputEntry struct {
	Node     string   `yaml:"node"`
	Statuses []string `yaml:"statuses"`
}

// codeStalenessOutputEntry mirrors codeStalenessEntry for YAML parsing.
type codeStalenessOutputEntry struct {
	Node   string `yaml:"node"`
	File   string `yaml:"file"`
	Status string `yaml:"status"`
}

// parseOutput parses the YAML stdout from the binary into toolOutput.
func parseOutput(t *testing.T, stdout string) toolOutput {
	t.Helper()

	var out toolOutput
	if err := yaml.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse YAML output: %v\nraw output:\n%s", err, stdout)
	}

	return out
}

// setupProjectRoot creates a t.TempDir() with a code-from-spec/ subdirectory
// and returns the path to the temp root (project root) and the code-from-spec/
// directory. The binary is always invoked from the project root.
func setupProjectRoot(t *testing.T) (projectRoot string, cfsDir string) {
	t.Helper()

	projectRoot = t.TempDir()
	cfsDir = filepath.Join(projectRoot, "code-from-spec")
	if err := os.MkdirAll(cfsDir, 0755); err != nil {
		t.Fatalf("failed to create code-from-spec directory: %v", err)
	}

	return projectRoot, cfsDir
}

// ---------------------------------------------------------------------------
// Help Message Tests
// ---------------------------------------------------------------------------

// TestHelp_WithHelpFlag verifies that passing --help prints the help message
// and exits with code 0. (Spec: "Any argument prints help")
func TestHelp_WithHelpFlag(t *testing.T) {
	// The help message does not require any spec tree, so we use a minimal
	// temp directory. However, the binary just checks len(os.Args) > 1.
	projectRoot, _ := setupProjectRoot(t)

	stdout, _, exitCode := runBinary(t, projectRoot, "--help")

	// Expect exit code 0.
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// Expect stdout to contain "staleness-check" and "Usage".
	if !strings.Contains(stdout, "staleness-check") {
		t.Errorf("expected stdout to contain 'staleness-check', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("expected stdout to contain 'Usage', got:\n%s", stdout)
	}
}

// TestHelp_WithArbitraryArg verifies that passing an arbitrary argument also
// prints the help message and exits with code 0. (Spec: "Different argument
// prints help")
func TestHelp_WithArbitraryArg(t *testing.T) {
	projectRoot, _ := setupProjectRoot(t)

	stdout, _, exitCode := runBinary(t, projectRoot, "foo")

	// Expect exit code 0.
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// Expect stdout to contain "staleness-check" and "Usage".
	if !strings.Contains(stdout, "staleness-check") {
		t.Errorf("expected stdout to contain 'staleness-check', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("expected stdout to contain 'Usage', got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// Happy Path Tests
// ---------------------------------------------------------------------------

// TestHappy_AllNodesUpToDate creates a minimal spec tree with no implements
// and no dependencies, verifying that the tool reports no problems and exits 0.
func TestHappy_AllNodesUpToDate(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Create the root node: spec/_node.md with version=1, title=ROOT.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Create a child node: spec/domain/_node.md with version=1,
	// parent_version=1, title=ROOT/domain.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
	}, "ROOT/domain")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 0 — no problems.
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout:\n%s", exitCode, stdout)
	}

	// Parse the output and verify all sections are empty.
	out := parseOutput(t, stdout)
	if len(out.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", out.SpecStaleness)
	}
	if len(out.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", out.TestStaleness)
	}
	if len(out.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", out.CodeStaleness)
	}
}

// TestHappy_NodeWithUpToDateGeneratedFile creates a node with an implements
// field pointing to a generated file whose spec comment matches the node's
// version. Expects all sections empty and exit code 0.
func TestHappy_NodeWithUpToDateGeneratedFile(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Child node with implements pointing to a generated file.
	// The implements path must be resolvable from the project root (the cwd).
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        2,
		"parent_version": 1,
		"implements":     []string{"gen.go"},
	}, "ROOT/domain")

	// Create the generated file at project root gen.go with matching version.
	writeGeneratedFile(t, projectRoot, "gen.go", "// spec: ROOT/domain@v2")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 0 — all up to date.
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)
	if len(out.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", out.SpecStaleness)
	}
	if len(out.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", out.TestStaleness)
	}
	if len(out.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", out.CodeStaleness)
	}
}

// TestHappy_NodeWithDependenciesAllCurrent creates a node that depends on
// another node, where the dependency version matches. Expects no problems.
func TestHappy_NodeWithDependenciesAllCurrent(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Domain node with version=3.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        3,
		"parent_version": 1,
	}, "ROOT/domain")

	// Config node depends on domain at version 3 (which matches).
	writeNodeFile(t, cfsDir, "spec/domain/config/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 3,
		"depends_on": []map[string]interface{}{
			{"path": "ROOT/domain", "version": 3},
		},
	}, "ROOT/domain/config")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 0 — all dependencies satisfied.
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)
	if len(out.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", out.SpecStaleness)
	}
	if len(out.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", out.TestStaleness)
	}
	if len(out.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", out.CodeStaleness)
	}
}

// ---------------------------------------------------------------------------
// Spec Staleness Tests
// ---------------------------------------------------------------------------

// TestSpecStaleness_ParentChanged creates a child node whose parent_version
// does not match the parent's actual version. Expects parent_changed status.
func TestSpecStaleness_ParentChanged(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node with version=2.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 2,
	}, "ROOT")

	// Domain node with parent_version=1, but parent is at version=2.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
	}, "ROOT/domain")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect one entry in spec_staleness for ROOT/domain with parent_changed.
	if len(out.SpecStaleness) != 1 {
		t.Fatalf("expected 1 spec_staleness entry, got %d: %v", len(out.SpecStaleness), out.SpecStaleness)
	}
	entry := out.SpecStaleness[0]
	if entry.Node != "ROOT/domain" {
		t.Errorf("expected node 'ROOT/domain', got '%s'", entry.Node)
	}
	if !containsStatus(entry.Statuses, "parent_changed") {
		t.Errorf("expected statuses to include 'parent_changed', got %v", entry.Statuses)
	}
}

// TestSpecStaleness_MultipleStatuses creates a node with wrong title, parent
// changed, and an invalid dependency. Expects all three statuses on one entry.
func TestSpecStaleness_MultipleStatuses(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node with version=2.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 2,
	}, "ROOT")

	// Domain node with:
	// - Wrong title (ROOT/domain/wrong instead of ROOT/domain) -> wrong_name
	// - parent_version=1 but parent is at version=2 -> parent_changed
	// - depends_on a non-existent node ROOT/missing -> invalid_dependency
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
		"depends_on": []map[string]interface{}{
			{"path": "ROOT/missing", "version": 1},
		},
	}, "ROOT/domain/wrong")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect one entry in spec_staleness for ROOT/domain.
	if len(out.SpecStaleness) != 1 {
		t.Fatalf("expected 1 spec_staleness entry, got %d: %v", len(out.SpecStaleness), out.SpecStaleness)
	}
	entry := out.SpecStaleness[0]
	if entry.Node != "ROOT/domain" {
		t.Errorf("expected node 'ROOT/domain', got '%s'", entry.Node)
	}

	// Verify all three statuses are present.
	if !containsStatus(entry.Statuses, "wrong_name") {
		t.Errorf("expected statuses to include 'wrong_name', got %v", entry.Statuses)
	}
	if !containsStatus(entry.Statuses, "parent_changed") {
		t.Errorf("expected statuses to include 'parent_changed', got %v", entry.Statuses)
	}
	if !containsStatus(entry.Statuses, "invalid_dependency") {
		t.Errorf("expected statuses to include 'invalid_dependency', got %v", entry.Statuses)
	}
}

// ---------------------------------------------------------------------------
// Test Staleness Tests
// ---------------------------------------------------------------------------

// TestTestStaleness_ParentChanged creates a test node whose parent_version
// does not match the parent spec node's version. Expects parent_changed in
// test_staleness.
func TestTestStaleness_ParentChanged(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Domain node with version=2.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        2,
		"parent_version": 1,
	}, "ROOT/domain")

	// Test node: parent_version=1 but parent (domain) is at version=2.
	writeNodeFile(t, cfsDir, "spec/domain/default.test.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
	}, "TEST/domain")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect one entry in test_staleness for TEST/domain with parent_changed.
	if len(out.TestStaleness) != 1 {
		t.Fatalf("expected 1 test_staleness entry, got %d: %v", len(out.TestStaleness), out.TestStaleness)
	}
	entry := out.TestStaleness[0]
	if entry.Node != "TEST/domain" {
		t.Errorf("expected node 'TEST/domain', got '%s'", entry.Node)
	}
	if !containsStatus(entry.Statuses, "parent_changed") {
		t.Errorf("expected statuses to include 'parent_changed', got %v", entry.Statuses)
	}
}

// ---------------------------------------------------------------------------
// Code Staleness Tests
// ---------------------------------------------------------------------------

// TestCodeStaleness_GeneratedFileStale creates a node at version 3 with a
// generated file that has spec comment at v2. Expects stale status.
func TestCodeStaleness_GeneratedFileStale(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Domain node at version=3 with implements.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        3,
		"parent_version": 1,
		"implements":     []string{"gen.go"},
	}, "ROOT/domain")

	// Generated file with stale spec comment (v2 instead of v3).
	writeGeneratedFile(t, projectRoot, "gen.go", "// spec: ROOT/domain@v2")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect one entry in code_staleness with status=stale.
	if len(out.CodeStaleness) != 1 {
		t.Fatalf("expected 1 code_staleness entry, got %d: %v", len(out.CodeStaleness), out.CodeStaleness)
	}
	entry := out.CodeStaleness[0]
	if entry.Node != "ROOT/domain" {
		t.Errorf("expected node 'ROOT/domain', got '%s'", entry.Node)
	}
	if !strings.Contains(entry.File, "gen.go") {
		t.Errorf("expected file containing 'gen.go', got '%s'", entry.File)
	}
	if entry.Status != "stale" {
		t.Errorf("expected status 'stale', got '%s'", entry.Status)
	}
}

// TestCodeStaleness_GeneratedFileMissing creates a node that implements a file
// which does not exist. Expects missing status.
func TestCodeStaleness_GeneratedFileMissing(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 1,
	}, "ROOT")

	// Domain node with implements pointing to a nonexistent file.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
		"implements":     []string{"nonexistent.go"},
	}, "ROOT/domain")

	// Do NOT create the file.

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect one entry in code_staleness with status=missing.
	if len(out.CodeStaleness) != 1 {
		t.Fatalf("expected 1 code_staleness entry, got %d: %v", len(out.CodeStaleness), out.CodeStaleness)
	}
	entry := out.CodeStaleness[0]
	if entry.Status != "missing" {
		t.Errorf("expected status 'missing', got '%s'", entry.Status)
	}
}

// ---------------------------------------------------------------------------
// Mixed Results Tests
// ---------------------------------------------------------------------------

// TestMixed_SpecTestAndCodeStalenessTogether creates a scenario where all three
// staleness sections have entries: spec staleness (parent_changed), test
// staleness (parent_changed), and code staleness (stale). Verifies exit code 1
// and all three sections populated.
func TestMixed_SpecTestAndCodeStalenessTogether(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node with version=2.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 2,
	}, "ROOT")

	// Domain node: version=3, parent_version=1 (parent is at 2 -> parent_changed).
	// Also implements gen.go.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        3,
		"parent_version": 1,
		"implements":     []string{"gen.go"},
	}, "ROOT/domain")

	// Test node: parent_version=1 but parent (domain) is at version=3 -> parent_changed.
	// Also implements gen_test.go.
	writeNodeFile(t, cfsDir, "spec/domain/default.test.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
		"implements":     []string{"gen_test.go"},
	}, "TEST/domain")

	// gen.go has v2 but node is at v3 -> stale.
	writeGeneratedFile(t, projectRoot, "gen.go", "// spec: ROOT/domain@v2")

	// gen_test.go has v1 which matches the test node version -> up to date.
	writeGeneratedFile(t, projectRoot, "gen_test.go", "// spec: TEST/domain@v1")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Spec staleness: ROOT/domain has parent_changed (parent_version=1 vs parent version=2).
	if len(out.SpecStaleness) < 1 {
		t.Fatalf("expected at least 1 spec_staleness entry, got %d", len(out.SpecStaleness))
	}
	foundSpecDomain := false
	for _, e := range out.SpecStaleness {
		if e.Node == "ROOT/domain" {
			foundSpecDomain = true
			if !containsStatus(e.Statuses, "parent_changed") {
				t.Errorf("expected ROOT/domain spec_staleness to include 'parent_changed', got %v", e.Statuses)
			}
		}
	}
	if !foundSpecDomain {
		t.Errorf("expected spec_staleness to contain ROOT/domain entry")
	}

	// Test staleness: TEST/domain has parent_changed (parent_version=1 vs parent version=3).
	if len(out.TestStaleness) < 1 {
		t.Fatalf("expected at least 1 test_staleness entry, got %d", len(out.TestStaleness))
	}
	foundTestDomain := false
	for _, e := range out.TestStaleness {
		if e.Node == "TEST/domain" {
			foundTestDomain = true
			if !containsStatus(e.Statuses, "parent_changed") {
				t.Errorf("expected TEST/domain test_staleness to include 'parent_changed', got %v", e.Statuses)
			}
		}
	}
	if !foundTestDomain {
		t.Errorf("expected test_staleness to contain TEST/domain entry")
	}

	// Code staleness: gen.go is stale (v2 vs v3). gen_test.go is up to date.
	if len(out.CodeStaleness) < 1 {
		t.Fatalf("expected at least 1 code_staleness entry, got %d", len(out.CodeStaleness))
	}
	foundStaleGen := false
	for _, e := range out.CodeStaleness {
		if strings.Contains(e.File, "gen.go") && !strings.Contains(e.File, "gen_test.go") {
			foundStaleGen = true
			if e.Status != "stale" {
				t.Errorf("expected gen.go status 'stale', got '%s'", e.Status)
			}
		}
	}
	if !foundStaleGen {
		t.Errorf("expected code_staleness to contain a stale gen.go entry")
	}
}

// ---------------------------------------------------------------------------
// Operational Error Tests
// ---------------------------------------------------------------------------

// TestOperationalError_SpecDirMissing creates a code-from-spec/ directory
// without a spec/ subdirectory. Expects exit code 2 and error on stderr.
func TestOperationalError_SpecDirMissing(t *testing.T) {
	projectRoot, _ := setupProjectRoot(t)

	// Do NOT create spec/ directory inside code-from-spec/.

	_, stderr, exitCode := runBinary(t, projectRoot)

	// Expect exit code 2 — operational error.
	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}

	// Expect stderr to contain an error message.
	if stderr == "" {
		t.Errorf("expected error message on stderr, got empty string")
	}
}

// ---------------------------------------------------------------------------
// Output Ordering Tests
// ---------------------------------------------------------------------------

// TestOutputOrdering_NodesSortedAlphabetically creates two nodes that both
// have parent_changed, and verifies they appear in alphabetical order in
// the spec_staleness output.
func TestOutputOrdering_NodesSortedAlphabetically(t *testing.T) {
	projectRoot, cfsDir := setupProjectRoot(t)

	// Root node with version=2.
	writeNodeFile(t, cfsDir, "spec/_node.md", map[string]interface{}{
		"version": 2,
	}, "ROOT")

	// Domain node: parent_version=1 but parent is at 2 -> parent_changed.
	writeNodeFile(t, cfsDir, "spec/domain/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
	}, "ROOT/domain")

	// Arch node: parent_version=1 but parent is at 2 -> parent_changed.
	writeNodeFile(t, cfsDir, "spec/arch/_node.md", map[string]interface{}{
		"version":        1,
		"parent_version": 1,
	}, "ROOT/arch")

	stdout, _, exitCode := runBinary(t, projectRoot)

	// Expect exit code 1 — problems found.
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	out := parseOutput(t, stdout)

	// Expect at least 2 entries in spec_staleness.
	if len(out.SpecStaleness) < 2 {
		t.Fatalf("expected at least 2 spec_staleness entries, got %d: %v", len(out.SpecStaleness), out.SpecStaleness)
	}

	// Verify ROOT/arch comes before ROOT/domain in the output.
	archIdx := -1
	domainIdx := -1
	for i, e := range out.SpecStaleness {
		if e.Node == "ROOT/arch" {
			archIdx = i
		}
		if e.Node == "ROOT/domain" {
			domainIdx = i
		}
	}

	if archIdx < 0 {
		t.Fatalf("expected to find ROOT/arch in spec_staleness")
	}
	if domainIdx < 0 {
		t.Fatalf("expected to find ROOT/domain in spec_staleness")
	}
	if archIdx >= domainIdx {
		t.Errorf("expected ROOT/arch (index %d) before ROOT/domain (index %d)", archIdx, domainIdx)
	}
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

// containsStatus checks whether a status string is present in a slice of
// statuses. Used for verifying that expected statuses appear in output.
func containsStatus(statuses []string, target string) bool {
	for _, s := range statuses {
		if s == target {
			return true
		}
	}
	return false
}
