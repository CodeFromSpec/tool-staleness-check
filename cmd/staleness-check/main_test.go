// code-from-spec: TEST/tech_design/main@v9
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// binaryPath holds the absolute path to the compiled staleness-check binary.
// Set once in TestMain and reused by every test.
var binaryPath string

// TestMain builds the binary once before any tests run.
func TestMain(m *testing.M) {
	// Build the binary into a temporary directory.
	tmpDir, err := os.MkdirTemp("", "staleness-check-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir for binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "staleness-check.exe")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// --- Helper functions ---

// runBinary executes the staleness-check binary in the given directory with the given args.
// Returns stdout, stderr, and the exit code.
func runBinary(t *testing.T, dir string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir

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

// createNodeFile creates a _node.md file at the given path (relative to root)
// with the specified frontmatter fields and title.
func createNodeFile(t *testing.T, root string, relPath string, fm map[string]interface{}, title string) {
	t.Helper()
	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", relPath, err)
	}

	// Marshal the frontmatter to YAML.
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		t.Fatalf("failed to marshal frontmatter for %s: %v", relPath, err)
	}

	// Build the file content: frontmatter + title.
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(fmBytes)
	content.WriteString("---\n\n")
	if title != "" {
		content.WriteString("# " + title + "\n")
	}

	if err := os.WriteFile(absPath, []byte(content.String()), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

// createGeneratedFile creates a generated source file with the given spec comment.
func createGeneratedFile(t *testing.T, root string, relPath string, specComment string) {
	t.Helper()
	absPath := filepath.Join(root, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", relPath, err)
	}

	content := specComment + "\npackage main\n"
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", relPath, err)
	}
}

// output is the structure used to parse the YAML output from the binary.
type output struct {
	SpecStaleness []specEntry `yaml:"spec_staleness"`
	TestStaleness []specEntry `yaml:"test_staleness"`
	CodeStaleness []codeEntry `yaml:"code_staleness"`
}

type specEntry struct {
	Node     string   `yaml:"node"`
	Statuses []string `yaml:"statuses"`
}

type codeEntry struct {
	Node   string `yaml:"node"`
	File   string `yaml:"file"`
	Status string `yaml:"status"`
}

// parseOutput parses the YAML output from the binary.
func parseOutput(t *testing.T, raw string) output {
	t.Helper()
	var o output
	if err := yaml.Unmarshal([]byte(raw), &o); err != nil {
		t.Fatalf("failed to parse YAML output: %v\nraw output:\n%s", err, raw)
	}
	return o
}

// containsStatus checks if a list of statuses contains a specific status.
func containsStatus(statuses []string, status string) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

// --- Help Message Tests ---

func TestHelpWithHelpFlag(t *testing.T) {
	// Any argument prints help. Invoke with --help.
	// Expect exit code 0 and stdout containing "staleness-check" and "Usage".
	dir := t.TempDir()
	stdout, _, exitCode := runBinary(t, dir, "--help")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "staleness-check") {
		t.Error("expected stdout to contain 'staleness-check'")
	}
	if !strings.Contains(stdout, "Usage") {
		t.Error("expected stdout to contain 'Usage'")
	}
}

func TestHelpWithArbitraryArg(t *testing.T) {
	// Different argument prints help. Invoke with "foo".
	// Expect exit code 0 and stdout containing "staleness-check" and "Usage".
	dir := t.TempDir()
	stdout, _, exitCode := runBinary(t, dir, "foo")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout, "staleness-check") {
		t.Error("expected stdout to contain 'staleness-check'")
	}
	if !strings.Contains(stdout, "Usage") {
		t.Error("expected stdout to contain 'Usage'")
	}
}

// --- Happy Path Tests ---

func TestAllNodesUpToDate(t *testing.T) {
	// Minimal spec tree with no staleness issues.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{"version": 1, "parent_version": 1},
		"ROOT/domain",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", o.SpecStaleness)
	}
	if len(o.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", o.TestStaleness)
	}
	if len(o.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", o.CodeStaleness)
	}
}

func TestNodeWithUpToDateGeneratedFile(t *testing.T) {
	// Node with implements list and matching spec comment in generated file.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{
			"version":        2,
			"parent_version": 1,
			"implements":     []string{"cmd/staleness-check/gen.go"},
		},
		"ROOT/domain",
	)
	createGeneratedFile(t, root, "cmd/staleness-check/gen.go",
		"// code-from-spec: ROOT/domain@v2",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", o.SpecStaleness)
	}
	if len(o.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", o.TestStaleness)
	}
	if len(o.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", o.CodeStaleness)
	}
}

func TestNodeWithDependenciesAllCurrent(t *testing.T) {
	// Node depends on another node; dependency version matches.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{"version": 3, "parent_version": 1},
		"ROOT/domain",
	)
	createNodeFile(t, root, "code-from-spec/domain/config/_node.md",
		map[string]interface{}{
			"version":        1,
			"parent_version": 3,
			"depends_on": []map[string]interface{}{
				{"path": "ROOT/domain", "version": 3},
			},
		},
		"ROOT/domain/config",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) != 0 {
		t.Errorf("expected empty spec_staleness, got %v", o.SpecStaleness)
	}
	if len(o.TestStaleness) != 0 {
		t.Errorf("expected empty test_staleness, got %v", o.TestStaleness)
	}
	if len(o.CodeStaleness) != 0 {
		t.Errorf("expected empty code_staleness, got %v", o.CodeStaleness)
	}
}

// --- Spec Staleness Tests ---

func TestParentChanged(t *testing.T) {
	// parent_version=1 but parent's actual version=2.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 2},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{"version": 1, "parent_version": 1},
		"ROOT/domain",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) == 0 {
		t.Fatal("expected non-empty spec_staleness")
	}

	// Find the entry for ROOT/domain.
	found := false
	for _, entry := range o.SpecStaleness {
		if entry.Node == "ROOT/domain" {
			found = true
			if !containsStatus(entry.Statuses, "parent_changed") {
				t.Errorf("expected 'parent_changed' in statuses, got %v", entry.Statuses)
			}
		}
	}
	if !found {
		t.Error("expected entry for ROOT/domain in spec_staleness")
	}
}

func TestMultipleStatusesOnOneNode(t *testing.T) {
	// Wrong title, parent changed, invalid dependency — all on one node.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 2},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{
			"version":        1,
			"parent_version": 1,
			"depends_on": []map[string]interface{}{
				{"path": "ROOT/missing", "version": 1},
			},
		},
		// Wrong title: should be ROOT/domain but is ROOT/domain/wrong.
		"ROOT/domain/wrong",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) == 0 {
		t.Fatal("expected non-empty spec_staleness")
	}

	found := false
	for _, entry := range o.SpecStaleness {
		if entry.Node == "ROOT/domain" {
			found = true
			if !containsStatus(entry.Statuses, "wrong_name") {
				t.Errorf("expected 'wrong_name' in statuses, got %v", entry.Statuses)
			}
			if !containsStatus(entry.Statuses, "parent_changed") {
				t.Errorf("expected 'parent_changed' in statuses, got %v", entry.Statuses)
			}
			if !containsStatus(entry.Statuses, "invalid_dependency") {
				t.Errorf("expected 'invalid_dependency' in statuses, got %v", entry.Statuses)
			}
		}
	}
	if !found {
		t.Error("expected entry for ROOT/domain in spec_staleness")
	}
}

// --- Test Staleness Tests ---

func TestTestNodeSubjectChanged(t *testing.T) {
	// Test node's subject_version=1 but subject's actual version=2.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{"version": 2, "parent_version": 1},
		"ROOT/domain",
	)
	createNodeFile(t, root, "code-from-spec/domain/default.test.md",
		map[string]interface{}{"version": 1, "subject_version": 1},
		"TEST/domain",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.TestStaleness) == 0 {
		t.Fatal("expected non-empty test_staleness")
	}

	found := false
	for _, entry := range o.TestStaleness {
		if entry.Node == "TEST/domain" {
			found = true
			if !containsStatus(entry.Statuses, "subject_changed") {
				t.Errorf("expected 'subject_changed' in statuses, got %v", entry.Statuses)
			}
		}
	}
	if !found {
		t.Error("expected entry for TEST/domain in test_staleness")
	}
}

// --- Code Staleness Tests ---

func TestGeneratedFileIsStale(t *testing.T) {
	// Node version=3 but generated file has spec comment v2.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{
			"version":        3,
			"parent_version": 1,
			"implements":     []string{"cmd/staleness-check/gen.go"},
		},
		"ROOT/domain",
	)
	createGeneratedFile(t, root, "cmd/staleness-check/gen.go",
		"// code-from-spec: ROOT/domain@v2",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.CodeStaleness) == 0 {
		t.Fatal("expected non-empty code_staleness")
	}

	found := false
	for _, entry := range o.CodeStaleness {
		if entry.Node == "ROOT/domain" && strings.Contains(entry.File, "gen.go") {
			found = true
			if entry.Status != "stale" {
				t.Errorf("expected status 'stale', got %q", entry.Status)
			}
		}
	}
	if !found {
		t.Error("expected code_staleness entry for ROOT/domain gen.go")
	}
}

func TestGeneratedFileMissing(t *testing.T) {
	// Node has implements but the file does not exist.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 1},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{
			"version":        1,
			"parent_version": 1,
			"implements":     []string{"cmd/staleness-check/nonexistent.go"},
		},
		"ROOT/domain",
	)
	// Do NOT create the file.

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.CodeStaleness) == 0 {
		t.Fatal("expected non-empty code_staleness")
	}

	found := false
	for _, entry := range o.CodeStaleness {
		if entry.Status == "missing" {
			found = true
		}
	}
	if !found {
		t.Error("expected code_staleness entry with status 'missing'")
	}
}

// --- Mixed Results Tests ---

func TestSpecTestAndCodeStalenessTogether(t *testing.T) {
	// All three sections have entries simultaneously.
	root := t.TempDir()

	// ROOT has version=2.
	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 2},
		"ROOT",
	)
	// ROOT/domain has parent_version=1 (parent is 2) → parent_changed.
	// Implements gen.go at version 3.
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{
			"version":        3,
			"parent_version": 1,
			"implements":     []string{"cmd/staleness-check/gen.go"},
		},
		"ROOT/domain",
	)
	// TEST/domain has subject_version=1 (subject is 3) → subject_changed.
	// Implements gen_test.go at version 1.
	createNodeFile(t, root, "code-from-spec/domain/default.test.md",
		map[string]interface{}{
			"version":         1,
			"subject_version": 1,
			"implements":      []string{"cmd/staleness-check/gen_test.go"},
		},
		"TEST/domain",
	)
	// gen.go has v2 but node is v3 → stale.
	createGeneratedFile(t, root, "cmd/staleness-check/gen.go",
		"// code-from-spec: ROOT/domain@v2",
	)
	// gen_test.go has v1 and test node is v1 → up to date.
	createGeneratedFile(t, root, "cmd/staleness-check/gen_test.go",
		"// code-from-spec: TEST/domain@v1",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)

	// Spec staleness: ROOT/domain should have parent_changed.
	if len(o.SpecStaleness) == 0 {
		t.Error("expected non-empty spec_staleness")
	} else {
		found := false
		for _, entry := range o.SpecStaleness {
			if entry.Node == "ROOT/domain" {
				found = true
				if !containsStatus(entry.Statuses, "parent_changed") {
					t.Errorf("expected 'parent_changed' in statuses, got %v", entry.Statuses)
				}
			}
		}
		if !found {
			t.Error("expected spec_staleness entry for ROOT/domain")
		}
	}

	// Test staleness: TEST/domain should have subject_changed.
	if len(o.TestStaleness) == 0 {
		t.Error("expected non-empty test_staleness")
	} else {
		found := false
		for _, entry := range o.TestStaleness {
			if entry.Node == "TEST/domain" {
				found = true
				if !containsStatus(entry.Statuses, "subject_changed") {
					t.Errorf("expected 'subject_changed' in statuses, got %v", entry.Statuses)
				}
			}
		}
		if !found {
			t.Error("expected test_staleness entry for TEST/domain")
		}
	}

	// Code staleness: gen.go should be stale. gen_test.go should NOT appear.
	if len(o.CodeStaleness) == 0 {
		t.Error("expected non-empty code_staleness")
	} else {
		foundStale := false
		for _, entry := range o.CodeStaleness {
			if strings.Contains(entry.File, "gen.go") && !strings.Contains(entry.File, "gen_test.go") {
				foundStale = true
				if entry.Status != "stale" {
					t.Errorf("expected status 'stale' for gen.go, got %q", entry.Status)
				}
			}
			// gen_test.go should not appear (it is up to date).
			if strings.Contains(entry.File, "gen_test.go") {
				t.Errorf("gen_test.go should not appear in code_staleness, but found with status %q", entry.Status)
			}
		}
		if !foundStale {
			t.Error("expected code_staleness entry for gen.go with status 'stale'")
		}
	}
}

// --- Operational Error Tests ---

func TestCodeFromSpecDirectoryMissing(t *testing.T) {
	// No code-from-spec/ directory → exit code 2 with stderr message.
	root := t.TempDir()
	// Do not create code-from-spec/ subdirectory.

	_, stderr, exitCode := runBinary(t, root)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if stderr == "" {
		t.Error("expected non-empty stderr for operational error")
	}
}

// --- Output Ordering Tests ---

func TestNodesSortedAlphabetically(t *testing.T) {
	// Both ROOT/arch and ROOT/domain have parent_changed.
	// ROOT/arch should appear before ROOT/domain in output.
	root := t.TempDir()

	createNodeFile(t, root, "code-from-spec/_node.md",
		map[string]interface{}{"version": 2},
		"ROOT",
	)
	createNodeFile(t, root, "code-from-spec/domain/_node.md",
		map[string]interface{}{"version": 1, "parent_version": 1},
		"ROOT/domain",
	)
	createNodeFile(t, root, "code-from-spec/arch/_node.md",
		map[string]interface{}{"version": 1, "parent_version": 1},
		"ROOT/arch",
	)

	stdout, _, exitCode := runBinary(t, root)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	o := parseOutput(t, stdout)
	if len(o.SpecStaleness) < 2 {
		t.Fatalf("expected at least 2 spec_staleness entries, got %d", len(o.SpecStaleness))
	}

	// Find the indices of ROOT/arch and ROOT/domain.
	archIdx := -1
	domainIdx := -1
	for i, entry := range o.SpecStaleness {
		if entry.Node == "ROOT/arch" {
			archIdx = i
		}
		if entry.Node == "ROOT/domain" {
			domainIdx = i
		}
	}

	if archIdx == -1 {
		t.Error("expected ROOT/arch in spec_staleness")
	}
	if domainIdx == -1 {
		t.Error("expected ROOT/domain in spec_staleness")
	}
	if archIdx != -1 && domainIdx != -1 && archIdx >= domainIdx {
		t.Errorf("expected ROOT/arch (index %d) before ROOT/domain (index %d)", archIdx, domainIdx)
	}
}
