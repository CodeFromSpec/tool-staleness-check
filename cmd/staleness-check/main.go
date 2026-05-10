// code-from-spec: ROOT/tech_design/main@v21
//
// Entry point for the staleness-check CLI tool.
// Orchestrates discovery, frontmatter parsing, staleness
// verification, and YAML output.
//
// Execution flow (per ROOT/tech_design/main spec):
//  1. If any CLI argument is passed → print help and exit 0.
//  2. Discover all spec and test nodes via DiscoverNodes.
//  3. Build a frontmatter cache (file path → *Frontmatter, nil on failure).
//  4. Run spec staleness checks (sorted alphabetically by logical name).
//  5. Run test staleness checks (sorted alphabetically by logical name).
//  6. Run code staleness checks across all nodes (sorted alphabetically).
//  7. Emit YAML to stdout; exit 0 (clean), 1 (problems), or 2 (op error).
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/codestaleness"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/discovery"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/specstaleness"
	"github.com/goccy/go-yaml"
)

// helpMessage is printed verbatim when any argument is passed.
// The exact wording is prescribed by ROOT/tech_design/main — do not alter it.
const helpMessage = `staleness-check — verifies spec and code staleness for a Code from Spec project.

Usage: staleness-check

Run from the project root with no arguments.
Outputs YAML to stdout with three sections:

  spec_staleness:
    - node: <logical-name>
      statuses:
        - <status>
  test_staleness:
    - node: <logical-name>
      statuses:
        - <status>
  code_staleness:
    - node: <logical-name>
      file: <file-path>
      status: <status>

Sections with no problems are empty lists ([]).

Spec and test staleness statuses:
  invalid_frontmatter  Frontmatter cannot be parsed or is missing required fields.
  wrong_name           Title does not match expected logical name.
  invalid_parent       Parent file cannot be found or read. (spec nodes)
  parent_changed       Parent version changed. (spec nodes)
  invalid_subject      Subject file cannot be found or read. (test nodes)
  subject_changed      Subject version changed. (test nodes)
  invalid_dependency   Dependency is malformed or cannot be found or read.
  dependency_changed   Dependency version changed.

Code staleness statuses:
  unreadable_frontmatter  Frontmatter cannot be parsed.
  no_version              Frontmatter has no version field.
  missing                 File in implements does not exist.
  no_spec_comment         File exists but has no spec comment.
  malformed_spec_comment  Spec comment exists but cannot be parsed.
  wrong_node              Spec comment references a different node.
  stale                   Spec version differs from spec comment version.

Exit codes: 0 = no problems, 1 = problems found, 2 = operational error.`

// specStalenessEntry represents one spec or test node that has staleness problems.
// YAML field names are prescribed by ROOT/domain/output — must not be renamed.
type specStalenessEntry struct {
	Node     string   `yaml:"node"`
	Statuses []string `yaml:"statuses"`
}

// codeStalenessEntry represents one generated file with a staleness problem.
// YAML field names are prescribed by ROOT/domain/output — must not be renamed.
type codeStalenessEntry struct {
	Node   string `yaml:"node"`
	File   string `yaml:"file"`
	Status string `yaml:"status"`
}

// output is the top-level YAML document emitted to stdout.
// Three sections are always present (empty list when no problems), in order:
// spec_staleness → test_staleness → code_staleness.
// Field order in the struct determines YAML key order with go-yaml.
type output struct {
	SpecStaleness []specStalenessEntry `yaml:"spec_staleness"`
	TestStaleness []specStalenessEntry `yaml:"test_staleness"`
	CodeStaleness []codeStalenessEntry `yaml:"code_staleness"`
}

func main() {
	// If any argument is passed, print help to stdout and exit 0.
	// The spec says "any argument", so we do not inspect what was passed.
	if len(os.Args) > 1 {
		fmt.Println(helpMessage)
		os.Exit(0)
	}

	// Step 1: Discover all spec nodes (_node.md) and test nodes (*.test.md)
	// under the code-from-spec/ directory.
	specNodes, testNodes, err := discovery.DiscoverNodes()
	if err != nil {
		// DiscoverNodes failure is an operational error — stderr, exit 2.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Step 2: Build the frontmatter cache.
	// Every discovered node gets an entry: *Frontmatter on success, nil on failure.
	// Frontmatter parse failures are NOT operational errors; they surface later
	// as invalid_frontmatter statuses during staleness verification.
	cache := buildFrontmatterCache(specNodes, testNodes)

	// Step 3: Spec staleness — check each spec node, sorted by logical name.
	// DiscoverNodes already returns sorted slices, but we sort again for safety.
	sortNodesByName(specNodes)
	specResults := collectSpecStaleness(specNodes, cache)

	// Step 4: Test staleness — check each test node, sorted by logical name.
	sortNodesByName(testNodes)
	testResults := collectSpecStaleness(testNodes, cache)

	// Step 5: Code staleness — check all nodes (spec + test), sorted by logical name.
	// The combined slice is re-sorted so that interleaved ROOT/TEST names come out
	// in a single alphabetical order as required by the spec.
	allNodes := make([]discovery.DiscoveredNode, 0, len(specNodes)+len(testNodes))
	allNodes = append(allNodes, specNodes...)
	allNodes = append(allNodes, testNodes...)
	sortNodesByName(allNodes)
	codeResults := collectCodeStaleness(allNodes, cache)

	// Step 6: Emit YAML to stdout.
	out := output{
		SpecStaleness: specResults,
		TestStaleness: testResults,
		CodeStaleness: codeResults,
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		// Marshalling a well-typed struct should never fail, but the spec
		// requires that every error is handled — stderr, exit 2.
		fmt.Fprintf(os.Stderr, "Error: failed to marshal YAML output: %v\n", err)
		os.Exit(2)
	}

	fmt.Print(string(data))

	// Step 7: Exit with the appropriate code.
	//   0 — all three sections are empty (no problems).
	//   1 — at least one section has entries (problems found).
	if len(specResults) > 0 || len(testResults) > 0 || len(codeResults) > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

// buildFrontmatterCache parses frontmatter for every discovered node and
// returns a map keyed by file path. A nil value means parsing failed for
// that file — the key is still present so callers can distinguish "not
// discovered" from "discovered but unparseable".
func buildFrontmatterCache(
	specNodes, testNodes []discovery.DiscoveredNode,
) map[string]*frontmatter.Frontmatter {
	cache := make(map[string]*frontmatter.Frontmatter)

	for _, node := range specNodes {
		fm, err := frontmatter.ParseFrontmatter(node.FilePath)
		if err != nil {
			// Parse failure → store nil so callers know the file exists
			// but its frontmatter is unreadable.
			cache[node.FilePath] = nil
		} else {
			cache[node.FilePath] = fm
		}
	}

	for _, node := range testNodes {
		fm, err := frontmatter.ParseFrontmatter(node.FilePath)
		if err != nil {
			cache[node.FilePath] = nil
		} else {
			cache[node.FilePath] = fm
		}
	}

	return cache
}

// sortNodesByName sorts a slice of DiscoveredNode in-place, alphabetically
// by LogicalName. This ensures output sections are deterministic across runs.
func sortNodesByName(nodes []discovery.DiscoveredNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].LogicalName < nodes[j].LogicalName
	})
}

// collectSpecStaleness runs CheckSpecStaleness for each node in the slice
// and aggregates all returned statuses into a single entry per node.
// Nodes with no problems are omitted. Always returns a non-nil slice so
// that go-yaml emits "[]" rather than "null" for empty sections.
func collectSpecStaleness(
	nodes []discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []specStalenessEntry {
	// Pre-allocate as empty (not nil) so YAML serializes as [].
	entries := make([]specStalenessEntry, 0)

	for _, node := range nodes {
		results := specstaleness.CheckSpecStaleness(node, cache)
		if len(results) == 0 {
			// Node is clean — omit from output.
			continue
		}

		// Collect all status strings for this node into one entry.
		// The spec allows multiple statuses per node (e.g., wrong_name +
		// parent_changed + dependency_changed simultaneously).
		statuses := make([]string, 0, len(results))
		for _, r := range results {
			statuses = append(statuses, r.Status)
		}

		entries = append(entries, specStalenessEntry{
			Node:     node.LogicalName,
			Statuses: statuses,
		})
	}

	return entries
}

// collectCodeStaleness runs CheckCodeStaleness for each node in the slice
// and collects one entry per problematic file. Files that are up to date
// are omitted. Always returns a non-nil slice so that go-yaml emits "[]"
// rather than "null" for an empty section.
func collectCodeStaleness(
	nodes []discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []codeStalenessEntry {
	// Pre-allocate as empty (not nil) so YAML serializes as [].
	entries := make([]codeStalenessEntry, 0)

	for _, node := range nodes {
		results := codestaleness.CheckCodeStaleness(node, cache)
		// Each result represents one file with one status (sequential checks).
		for _, r := range results {
			entries = append(entries, codeStalenessEntry{
				Node:   r.Node,
				File:   r.File,
				Status: r.Status,
			})
		}
	}

	return entries
}
