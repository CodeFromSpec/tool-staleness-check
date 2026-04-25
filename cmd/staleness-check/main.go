// spec: ROOT/tech_design/main@v9
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/goccy/go-yaml"
)

// helpMessage is the exact help text prescribed by the spec (ROOT/tech_design/main).
// It is printed to stdout when the tool is invoked with any argument.
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
  invalid_parent       Parent file cannot be found or read.
  parent_changed       Parent version changed.
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

// specStalenessEntry represents a single spec or test staleness output entry.
// The YAML field names must match the format prescribed by ROOT/domain/output:
// "node" (string) and "statuses" (list of strings).
type specStalenessEntry struct {
	Node     string   `yaml:"node"`
	Statuses []string `yaml:"statuses"`
}

// codeStalenessEntry represents a single code staleness output entry.
// The YAML field names must match the format prescribed by ROOT/domain/output:
// "node" (string), "file" (string), and "status" (string).
type codeStalenessEntry struct {
	Node   string `yaml:"node"`
	File   string `yaml:"file"`
	Status string `yaml:"status"`
}

// output is the top-level YAML output structure. The three sections are emitted
// in order: spec_staleness, test_staleness, code_staleness. Empty sections must
// produce `[]` in YAML (not null), which requires initialized empty slices.
type output struct {
	SpecStaleness []specStalenessEntry `yaml:"spec_staleness"`
	TestStaleness []specStalenessEntry `yaml:"test_staleness"`
	CodeStaleness []codeStalenessEntry `yaml:"code_staleness"`
}

func main() {
	// --- Arguments check (ROOT/tech_design/main: Arguments). ---
	// If any argument is passed, print help message and exit 0.
	if len(os.Args) > 1 {
		fmt.Println(helpMessage)
		os.Exit(0)
	}

	// --- Step 1: Discover all nodes (ROOT/tech_design/main: Execution flow step 1). ---
	// DiscoverNodes returns spec nodes, test nodes, and external dependencies.
	// On failure, print error to stderr and exit 2 (operational error).
	specNodes, testNodes, externalDeps, err := DiscoverNodes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	// --- Step 2: Build the frontmatter cache (ROOT/tech_design/main: Execution flow step 2). ---
	// Call ParseFrontmatter for every discovered node (spec, test, and external).
	// Store *Frontmatter on success, nil on failure — do not abort on parse errors.
	// The cache is keyed by file path.
	cache := make(map[string]*Frontmatter)

	// Helper to populate the cache for a slice of discovered nodes.
	// On success, stores the parsed *Frontmatter. On failure, stores nil
	// explicitly so the staleness checks can distinguish "file exists but
	// frontmatter is bad" from "file does not exist" (missing key).
	populateCache := func(nodes []DiscoveredNode) {
		for _, node := range nodes {
			fm, parseErr := ParseFrontmatter(node.FilePath)
			if parseErr != nil {
				// Store nil — frontmatter parsing failed. This is not an
				// operational error; it will be surfaced as a status during
				// staleness verification.
				cache[node.FilePath] = nil
			} else {
				cache[node.FilePath] = fm
			}
		}
	}

	// Populate cache for all three categories of discovered nodes.
	populateCache(specNodes)
	populateCache(testNodes)
	populateCache(externalDeps)

	// --- Step 3: Run spec staleness (ROOT/tech_design/main: Execution flow step 3). ---
	// Call CheckSpecStaleness for each spec node, sorted alphabetically by
	// logical name (already sorted by DiscoverNodes). Collect all results.
	var specResults []StalenessResult
	for _, node := range specNodes {
		results := CheckSpecStaleness(node, cache)
		specResults = append(specResults, results...)
	}

	// --- Step 4: Run test staleness (ROOT/tech_design/main: Execution flow step 4). ---
	// Call CheckSpecStaleness for each test node (same function, different input).
	var testResults []StalenessResult
	for _, node := range testNodes {
		results := CheckSpecStaleness(node, cache)
		testResults = append(testResults, results...)
	}

	// --- Step 5: Run code staleness (ROOT/tech_design/main: Execution flow step 5). ---
	// Call CheckCodeStaleness for each spec and test node, sorted alphabetically
	// by logical name. We merge spec and test nodes into a single sorted list
	// to process them in the correct order.
	allNodes := make([]DiscoveredNode, 0, len(specNodes)+len(testNodes))
	allNodes = append(allNodes, specNodes...)
	allNodes = append(allNodes, testNodes...)

	// Sort the combined list alphabetically by logical name so the output
	// order is deterministic and matches the spec requirement.
	sort.Slice(allNodes, func(i, j int) bool {
		return allNodes[i].LogicalName < allNodes[j].LogicalName
	})

	var codeResults []StalenessResult
	for _, node := range allNodes {
		results := CheckCodeStaleness(node, cache)
		codeResults = append(codeResults, results...)
	}

	// --- Step 6: Build and emit YAML output (ROOT/tech_design/main: Execution flow step 6). ---
	// Group spec and test staleness results by node, collecting statuses into
	// a list. Code staleness results are already one-per-entry.

	// Initialize with empty slices so YAML produces [] instead of null.
	out := output{
		SpecStaleness: groupStalenessResults(specResults),
		TestStaleness: groupStalenessResults(testResults),
		CodeStaleness: buildCodeEntries(codeResults),
	}

	// Encode the output as YAML to stdout using github.com/goccy/go-yaml
	// (required by ROOT/tech_design Dependencies constraint).
	encoder := yaml.NewEncoder(os.Stdout)
	if err := encoder.Encode(&out); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to encode YAML output: %v\n", err)
		os.Exit(2)
	}
	if err := encoder.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to close YAML encoder: %v\n", err)
		os.Exit(2)
	}

	// --- Step 7: Exit with appropriate code (ROOT/tech_design/main: Execution flow step 7). ---
	// Exit 0 if all sections are empty, 1 if any section has entries.
	if len(out.SpecStaleness) > 0 || len(out.TestStaleness) > 0 || len(out.CodeStaleness) > 0 {
		os.Exit(1)
	}

	// All sections empty — exit 0 (no problems found).
	os.Exit(0)
}

// groupStalenessResults groups StalenessResult entries by Node, collecting all
// statuses for each node into a single specStalenessEntry. Multiple results
// with the same Node (e.g., wrong_name + parent_changed) become one entry with
// a multi-element statuses list.
//
// The input order is preserved for grouping — results for the same node are
// assumed to be contiguous (they come from processing nodes in sorted order).
// Returns an initialized empty slice (not nil) if there are no results, so
// YAML serialization produces [] instead of null.
func groupStalenessResults(results []StalenessResult) []specStalenessEntry {
	// Initialize to empty slice so YAML produces [] not null.
	entries := make([]specStalenessEntry, 0)

	if len(results) == 0 {
		return entries
	}

	// Group results by Node. Because nodes are processed in sorted order,
	// results for the same node appear contiguously. We use a simple loop
	// that tracks the current node.
	var current *specStalenessEntry

	for _, r := range results {
		if current != nil && current.Node == r.Node {
			// Same node — add status to existing entry.
			current.Statuses = append(current.Statuses, r.Status)
		} else {
			// Different node — finalize the previous entry (if any) and start a new one.
			if current != nil {
				entries = append(entries, *current)
			}
			current = &specStalenessEntry{
				Node:     r.Node,
				Statuses: []string{r.Status},
			}
		}
	}

	// Don't forget to append the last entry.
	if current != nil {
		entries = append(entries, *current)
	}

	return entries
}

// buildCodeEntries converts StalenessResult entries into codeStalenessEntry
// entries for YAML output. Each StalenessResult maps directly to one code
// staleness entry with node, file, and status fields.
//
// Returns an initialized empty slice (not nil) if there are no results, so
// YAML serialization produces [] instead of null.
func buildCodeEntries(results []StalenessResult) []codeStalenessEntry {
	// Initialize to empty slice so YAML produces [] not null.
	entries := make([]codeStalenessEntry, 0)

	for _, r := range results {
		entries = append(entries, codeStalenessEntry{
			Node:   r.Node,
			File:   r.File,
			Status: r.Status,
		})
	}

	return entries
}
