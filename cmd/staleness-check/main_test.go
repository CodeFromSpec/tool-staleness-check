// code-from-spec: TEST/tech_design/main@v13
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
// The binary is built from the current package directory (".").
func TestMain(m *testing.M) {
	// Create a temporary directory to hold the compiled binary.
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

	// Build the binary from the current package directory.
	// During tests, the working directory is the package directory, so "." is correct.
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
// Prefixed with "test" per test convention to avoid collisions with package-level names.
func testWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// testMakeNodeFile creates a spec or test node file at relPath within projectRoot.
// frontmatterLines are raw YAML lines inserted between the --- delimiters.
// title is the logical name written as the H1 heading (e.g., "ROOT/domain" or "TEST/domain").
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

// testMakeGeneratedFile creates a generated source file with a spec comment as its first line.
// specComment is the comment body (without the leading "// "), e.g. "code-from-spec: ROOT/domain@v2".
func testMakeGeneratedFile(t *testing.T, projectRoot, relPath, specComment string) {
	t.Helper()
	content := "// " + specComment + "\npackage main\n"
	testWriteFile(t, filepath.Join(projectRoot, relPath), content)
}

// ── Help Message ─────────────────────────────────────────────────────────────

// TestHelpFlag verifies that passing --help prints the help message to stdout and exits 0.
// Per spec: any argument triggers help output.
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
// Per spec: "With any argument (e.g., --help, -h, or anything else)".
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

// ── Happy Path ────────────────────────────────────────────────────────────────

// TestAllNodesUpToDate verifies the clean output when all nodes are current.
// Per spec: exit code 0, all three sections empty ([]).
func TestAllNodesUpToDate(t *testing.T) {
	dir := t.TempDir()

	// ROOT node at version=1, no parent (it is the root).
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	// ROOT/domain node at version=1, parent_version=1 (matches ROOT's version=1).
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/domain",
	)

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Expect all sections to be explicitly empty lists.
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
// Per spec: spec comment version matches node version → exit code 0, all sections empty.
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
	// Generated file with matching spec comment (v2 = node version).
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
// Per spec: depends_on version matches actual node version → exit code 0.
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
	// config depends on ROOT/domain at version 3 — which matches domain's current version.
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

// ── Spec Staleness ────────────────────────────────────────────────────────────

// TestParentChanged verifies that a node with a stale parent_version is flagged.
// Per spec: parent_version=1 but ROOT is version=2 → parent_changed status, exit code 1.
func TestParentChanged(t *testing.T) {
	dir := t.TempDir()

	// ROOT has been updated to version=2.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// domain still tracks parent_version=1, which is now stale.
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
// invalid_dependency are all reported for the same node in a single entry.
// Per spec: multiple statuses accumulate on one node rather than creating separate entries.
func TestMultipleStatusesOnOneNode(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// Three problems:
	//   1. Title says "ROOT/domain/wrong" but logical name derived from path is "ROOT/domain" → wrong_name
	//   2. parent_version=1 but ROOT is version=2 → parent_changed
	//   3. depends_on ROOT/missing which doesn't exist → invalid_dependency
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 1",
			"parent_version: 1",
			"depends_on:",
			"  - path: ROOT/missing",
			"    version: 1",
		},
		"ROOT/domain/wrong", // intentionally wrong title
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

// ── Test Staleness ────────────────────────────────────────────────────────────

// TestTestNodeSubjectChanged verifies that a test node is flagged when its
// subject (spec node) version has advanced beyond subject_version.
// Per spec: subject_version=1 but subject is version=2 → subject_changed, exit code 1.
func TestTestNodeSubjectChanged(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	// domain has been updated to version=2.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 2", "parent_version: 1"},
		"ROOT/domain",
	)
	// Test node still tracks subject_version=1, but domain is now at version=2.
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

// ── Code Staleness ────────────────────────────────────────────────────────────

// TestGeneratedFileIsStale verifies that a generated file referencing an older
// spec version is flagged as stale.
// Per spec: file says v2 but node is v3 → stale status, exit code 1.
func TestGeneratedFileIsStale(t *testing.T) {
	dir := t.TempDir()

	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 1"},
		"ROOT",
	)
	// domain is at version=3, implements gen.go.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 3",
			"parent_version: 1",
			"implements:",
			"  - cmd/staleness-check/gen.go",
		},
		"ROOT/domain",
	)
	// Generated file has spec comment at v2, but node is at v3 — stale.
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

// TestGeneratedFileMissing verifies that a file listed in implements but absent
// on disk is reported with status=missing.
// Per spec: file does not exist → missing status, exit code 1.
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
	// Intentionally do NOT create cmd/staleness-check/nonexistent.go.

	stdout, _, exitCode := runBinary(t, dir)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d\nstdout:\n%s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "missing") {
		t.Errorf("expected stdout to contain 'missing', got:\n%s", stdout)
	}
}

// ── Mixed Results ─────────────────────────────────────────────────────────────

// TestMixedResults verifies that spec, test, and code staleness are all reported
// together in the same run.
// Per spec: all three sections must have entries; gen_test.go remains up to date.
func TestMixedResults(t *testing.T) {
	dir := t.TempDir()

	// ROOT at version=2 — domain's parent_version=1 will be stale.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// domain at version=3, parent_version=1 (stale: ROOT is v2), implements gen.go.
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{
			"version: 3",
			"parent_version: 1", // stale: ROOT is now v2
			"implements:",
			"  - cmd/staleness-check/gen.go",
		},
		"ROOT/domain",
	)
	// Test node: subject_version=1 but domain is v3 → subject_changed.
	// Also implements gen_test.go (which will be up to date).
	testMakeNodeFile(t, dir, "code-from-spec/domain/default.test.md",
		[]string{
			"version: 1",
			"subject_version: 1", // stale: domain is v3
			"implements:",
			"  - cmd/staleness-check/gen_test.go",
		},
		"TEST/domain",
	)
	// gen.go spec comment says v2, but domain is v3 — stale.
	testMakeGeneratedFile(t, dir, "cmd/staleness-check/gen.go", "code-from-spec: ROOT/domain@v2")
	// gen_test.go spec comment says v1, and test node is v1 — up to date.
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
	// All three top-level section keys must be present.
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

// ── Operational Error ─────────────────────────────────────────────────────────

// TestMissingCodeFromSpecDir verifies that exit code 2 and a non-empty stderr
// error message are produced when the code-from-spec/ directory is absent.
// Per spec: DiscoverNodes failure → print to stderr, exit 2.
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

// ── Output Ordering ───────────────────────────────────────────────────────────

// TestNodesSortedAlphabetically verifies that staleness entries appear sorted
// by logical name (ROOT/arch before ROOT/domain).
// Per spec: nodes are processed sorted alphabetically by logical name.
func TestNodesSortedAlphabetically(t *testing.T) {
	dir := t.TempDir()

	// ROOT at version=2; both child nodes track parent_version=1 — both stale.
	testMakeNodeFile(t, dir, "code-from-spec/_node.md",
		[]string{"version: 2"},
		"ROOT",
	)
	// domain added first, but should appear second in output (alphabetical).
	testMakeNodeFile(t, dir, "code-from-spec/domain/_node.md",
		[]string{"version: 1", "parent_version: 1"},
		"ROOT/domain",
	)
	// arch added second, but should appear first in output (alphabetical).
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

	// ROOT/arch must appear before ROOT/domain in the output.
	archIdx := strings.Index(stdout, "ROOT/arch")
	domainIdx := strings.Index(stdout, "ROOT/domain")
	if archIdx >= domainIdx {
		t.Errorf("expected ROOT/arch to appear before ROOT/domain in output:\n%s", stdout)
	}
}
