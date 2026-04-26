// code-from-spec: TEST/tech_design/main@v10
// spec: TEST/tech_design/main@v10
//
// Integration tests for the staleness-check binary.
// The binary is built once in TestMain and reused across all tests.
// Each test creates its own temporary directory representing the project root.

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath holds the path to the compiled binary, set in TestMain.
var binaryPath string

// TestMain builds the binary once and runs all tests.
func TestMain(m *testing.M) {
	// Create a temporary file path for the binary.
	tmpDir, err := os.MkdirTemp("", "staleness-check-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir for binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Set the binary name, adding .exe on Windows.
	binaryName := "staleness-check"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath = filepath.Join(tmpDir, binaryName)

	// Build the binary from the current package.
	// The working directory during tests is the package directory,
	// so we build the current directory (".").
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// runBinary invokes the staleness-check binary with the given arguments from the
// specified working directory (project root). Returns stdout, stderr, and exit code.
func runBinary(t *testing.T, projectRoot string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = projectRoot

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("unexpected error running binary: %v", err)
		}
	} else {
		exitCode = 0
	}
	return stdout, stderr, exitCode
}

// testWriteFile writes content to a file, creating parent directories as needed.
func testWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// testMakeNodeFile creates a _node.md file at the given path within projectRoot.
// frontmatterLines are raw YAML lines inserted between the --- delimiters.
// title is the logical name written as the title line (e.g., "ROOT/domain").
func testMakeNodeFile(t *testing.T, projectRoot, relPath string, frontmatterLines []string, title string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("---\n")
	for _, line := range frontmatterLines {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("---\n")
	sb.WriteString("\n")
	sb.WriteString("# " + title + "\n")
	testWriteFile(t, filepath.Join(projectRoot, relPath), sb.String())
}

// testMakeGeneratedFile creates a generated file with a spec comment as its first line.
func testMakeGeneratedFile(t *testing.T, projectRoot, relPath, specComment string) {
	t.Helper()
	content := "// " + specComment + "\npackage main\n"
	testWriteFile(t, filepath.Join(projectRoot, relPath), content)
}

// ── Help Message ────────────────────────────────────────────────────────────

// TestHelpFlag verifies that passing --help prints the help message and exits 0.
func TestHelpFlag(t *testing.T) {
	dir := t.TempDir()
	stdout, _, exitCode := runBinary(t, dir, "--help")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "staleness-check") {
		t.Errorf("expected stdout to contain 'staleness-check', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("expected stdout to contain 'Usage', got:\n%s", stdout)
	}
}

// TestHelpArbitraryArg verifies that any argument (not just --help) prints help and exits 0.
func TestHelpArbitraryArg(t *testing.T) {
	dir := t.TempDir()
	stdout, _, exitCode := runBinary(t, dir, "foo")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "staleness-check") {
		t.Errorf("expected stdout to contain 'staleness-check', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("expected stdout to contain 'Usage', got:\n%s", stdout)
	}
}

// ── Happy Path ───────────────────────────────────────────────────────────────

// TestAllNodesUpToDate verifies the clean output when all nodes are current.
func TestAllNodesUpToDate(t *testing.T) {
	dir := t.TempDir()

	// ROOT node: version=1, no parent
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	// ROOT/domain node: version=1, parent_version=1
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/domain",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Expect all sections empty.
	expectedLines := []string{
		"spec_staleness: []",
		"test_staleness: []",
		"code_staleness: []",
	}
	for _, line := range expectedLines {
		if !strings.Contains(stdout, line) {
			t.Errorf("expected stdout to contain %q, got:\n%s", line, stdout)
		}
	}
}

// TestNodeWithUpToDateGeneratedFile verifies that a node with a current generated file
// produces no staleness entries.
func TestNodeWithUpToDateGeneratedFile(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 2",
			"parent_version: 1",
			"implements:",
			"  - cmd/staleness-check/gen.go",
		},
		"ROOT/domain",
	)
	// Generated file with matching spec comment.
	testMakeGeneratedFile(t, dir, "cmd/staleness-check/gen.go", "code-from-spec: ROOT/domain@v2")

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "spec_staleness: []") {
		t.Errorf("expected spec_staleness: [], got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "test_staleness: []") {
		t.Errorf("expected test_staleness: [], got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "code_staleness: []") {
		t.Errorf("expected code_staleness: [], got:\n%s", stdout)
	}
}

// TestNodeWithDependenciesAllCurrent verifies that a node with current dependencies
// produces no staleness entries.
func TestNodeWithDependenciesAllCurrent(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 3", "parent_version: 1"},
		"ROOT/domain",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/config/_node.md",
		[]string{
			"version: 1",
			"parent_version: 3",
			"depends_on:",
			"  - path: ROOT/domain",
			"    version: 3",
		},
		"ROOT/domain/config",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "spec_staleness: []") {
		t.Errorf("expected spec_staleness: [], got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "test_staleness: []") {
		t.Errorf("expected test_staleness: [], got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "code_staleness: []") {
		t.Errorf("expected code_staleness: [], got:\n%s", stdout)
	}
}

// ── Spec Staleness ───────────────────────────────────────────────────────────

// TestParentChanged verifies that a node with a stale parent_version is flagged.
func TestParentChanged(t *testing.T) {
	dir := t.TempDir()

	// ROOT has version=2 but domain still tracks parent_version=1.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/domain",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "ROOT/domain") {
		t.Errorf("expected stdout to contain 'ROOT/domain', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "parent_changed") {
		t.Errorf("expected stdout to contain 'parent_changed', got:\n%s", stdout)
	}
}

// TestMultipleStatusesOnOneNode verifies that wrong_name, parent_changed, and
// invalid_dependency are all reported for the same node.
func TestMultipleStatusesOnOneNode(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// Wrong title, parent_version=1 but parent is version=2, depends on missing node.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 1",
			"parent_version: 1",
			"depends_on:",
			"  - path: ROOT/missing",
			"    version: 1",
		},
		"ROOT/domain/wrong", // wrong title — should be ROOT/domain
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "ROOT/domain") {
		t.Errorf("expected stdout to contain 'ROOT/domain', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "wrong_name") {
		t.Errorf("expected stdout to contain 'wrong_name', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "parent_changed") {
		t.Errorf("expected stdout to contain 'parent_changed', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "invalid_dependency") {
		t.Errorf("expected stdout to contain 'invalid_dependency', got:\n%s", stdout)
	}
}

// ── Test Staleness ───────────────────────────────────────────────────────────

// TestTestNodeSubjectChanged verifies that a test node flagged when the subject version changed.
func TestTestNodeSubjectChanged(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	// domain at version 2.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 2", "parent_version: 1"},
		"ROOT/domain",
	)
	// Test node still tracks subject_version=1 but domain is at version=2.
	testMakeNodeFile(t, dir, "code-from-spec/domain/default.test.md",
		[]string{"version: 1", "subject_version: 1"},
		"TEST/domain",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "TEST/domain") {
		t.Errorf("expected stdout to contain 'TEST/domain', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "subject_changed") {
		t.Errorf("expected stdout to contain 'subject_changed', got:\n%s", stdout)
	}
}

// ── Code Staleness ───────────────────────────────────────────────────────────

// TestGeneratedFileIsStale verifies that a generated file with an older spec comment is flagged.
func TestGeneratedFileIsStale(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 3",
			"parent_version: 1",
			"implements:",
			"  - cmd/staleness-check/gen.go",
		},
		"ROOT/domain",
	)
	// Generated file says v2 but node is at version 3.
	testMakeGeneratedFile(t, dir, "cmd/staleness-check/gen.go", "code-from-spec: ROOT/domain@v2")

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "ROOT/domain") {
		t.Errorf("expected stdout to contain 'ROOT/domain', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "gen.go") {
		t.Errorf("expected stdout to contain 'gen.go', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "stale") {
		t.Errorf("expected stdout to contain 'stale', got:\n%s", stdout)
	}
}

// TestGeneratedFileMissing verifies that a missing implements file is reported.
func TestGeneratedFileMissing(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 1",
			"parent_version: 1",
			"implements:",
			"  - cmd/staleness-check/nonexistent.go",
		},
		"ROOT/domain",
	)
	// Do NOT create cmd/staleness-check/nonexistent.go.

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "missing") {
		t.Errorf("expected stdout to contain 'missing', got:\n%s", stdout)
	}
}

// ── Mixed Results ────────────────────────────────────────────────────────────

// TestMixedResults verifies that spec, test, and code staleness are all reported together.
func TestMixedResults(t *testing.T) {
	dir := t.TempDir()

	// ROOT at version 2 — domain's parent_version=1 will be stale.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// domain at version 3, implements gen.go.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 3",
			"parent_version: 1", // stale: parent is v2
			"implements:",
			"  - cmd/staleness-check/gen.go",
		},
		"ROOT/domain",
	)
	// Test node: subject_version=1 but domain is v3.
	testMakeNodeFile(t, dir, "code-from-spec/domain/default.test.md",
		[]string{
			"version: 1",
			"subject_version: 1", // stale: subject is v3
			"implements:",
			"  - cmd/staleness-check/gen_test.go",
		},
		"TEST/domain",
	)
	// gen.go says v2 but domain is v3 — stale.
	testMakeGeneratedFile(t, dir, "cmd/staleness-check/gen.go", "code-from-spec: ROOT/domain@v2")
	// gen_test.go says v1 and test node is v1 — up to date.
	testMakeGeneratedFile(t, dir, "cmd/staleness-check/gen_test.go", "code-from-spec: TEST/domain@v1")

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	// Spec staleness: ROOT/domain has parent_changed.
	if !strings.Contains(stdout, "parent_changed") {
		t.Errorf("expected stdout to contain 'parent_changed', got:\n%s", stdout)
	}
	// Test staleness: TEST/domain has subject_changed.
	if !strings.Contains(stdout, "subject_changed") {
		t.Errorf("expected stdout to contain 'subject_changed', got:\n%s", stdout)
	}
	// Code staleness: gen.go is stale (v2 vs v3).
	if !strings.Contains(stdout, "stale") {
		t.Errorf("expected stdout to contain 'stale', got:\n%s", stdout)
	}
	// All three top-level sections must appear.
	if !strings.Contains(stdout, "spec_staleness:") {
		t.Errorf("expected stdout to contain 'spec_staleness:', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "test_staleness:") {
		t.Errorf("expected stdout to contain 'test_staleness:', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "code_staleness:") {
		t.Errorf("expected stdout to contain 'code_staleness:', got:\n%s", stdout)
	}
}

// ── Operational Error ────────────────────────────────────────────────────────

// TestMissingCodeFromSpecDir verifies that exit code 2 and a stderr error are
// produced when the code-from-spec/ directory is absent.
func TestMissingCodeFromSpecDir(t *testing.T) {
	// An empty TempDir has no code-from-spec/ subdirectory.
	dir := t.TempDir()

	_, stderr, exitCode := runBinary(t, dir)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("expected non-empty stderr error message, got empty")
	}
}

// ── Output Ordering ──────────────────────────────────────────────────────────

// TestNodesSortedAlphabetically verifies that staleness entries appear sorted
// by logical name (ROOT/arch before ROOT/domain).
func TestNodesSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()

	// ROOT at version 2; both child nodes track parent_version=1 — both stale.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/domain",
	)
	testMakeNodeFile(t, dir, "code-from-spec/arch/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/arch",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}

	// Both nodes must appear in the output.
	if !strings.Contains(stdout, "ROOT/arch") {
		t.Errorf("expected stdout to contain 'ROOT/arch', got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ROOT/domain") {
		t.Errorf("expected stdout to contain 'ROOT/domain', got:\n%s", stdout)
	}

	// ROOT/arch must appear before ROOT/domain.
	archIdx := strings.Index(stdout, "ROOT/arch")
	domainIdx := strings.Index(stdout, "ROOT/domain")
	if archIdx >= domainIdx {
		t.Errorf("expected ROOT/arch to appear before ROOT/domain in output:\n%s", stdout)
	}
}
