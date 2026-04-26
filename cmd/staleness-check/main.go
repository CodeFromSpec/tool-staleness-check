// code-from-spec: ROOT/tech_design/main@v18
//
// Entry point for the staleness-check CLI tool.
// Orchestrates discovery, frontmatter parsing, staleness
// verification, and YAML output.
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

// helpMessage is printed when any argument is passed.
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

// specStalenessEntry represents one spec or test node with staleness problems.
// YAML field names match the output format prescribed by ROOT/domain/output.
type specStalenessEntry struct {
	Node     string   `yaml:"node"`
	Statuses []string `yaml:"statuses"`
}

// codeStalenessEntry represents one generated file with a staleness problem.
// YAML field names match the output format prescribed by ROOT/domain/output.
type codeStalenessEntry struct {
	Node   string `yaml:"node"`
	File   string `yaml:"file"`
	Status string `yaml:"status"`
}

// output is the top-level YAML structure emitted to stdout.
// Field order and names match ROOT/domain/output.
type output struct {
	SpecStaleness []specStalenessEntry `yaml:"spec_staleness"`
	TestStaleness []specStalenessEntry `yaml:"test_staleness"`
	CodeStaleness []codeStalenessEntry `yaml:"code_staleness"`
}

func main() {
	// If any argument is passed, print help and exit 0.
	if len(os.Args) > 1 {
		fmt.Println(helpMessage)
		os.Exit(0)
	}

	// Step 1: Discover all spec and test nodes.
	specNodes, testNodes, err := discovery.DiscoverNodes()
	if err != nil {
		// Operational error — print to stderr and exit 2.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Step 2: Build the frontmatter cache.
	// Maps file path → *Frontmatter (nil on parse failure).
	cache := buildFrontmatterCache(specNodes, testNodes)

	// Step 3: Run spec staleness for each spec node, sorted by logical name.
	// specNodes are already sorted by DiscoverNodes, but we sort again to be safe.
	sortNodesByName(specNodes)
	specResults := collectSpecStaleness(specNodes, cache)

	// Step 4: Run test staleness for each test node, sorted by logical name.
	sortNodesByName(testNodes)
	testResults := collectSpecStaleness(testNodes, cache)

	// Step 5: Run code staleness for all nodes (spec + test), sorted by logical name.
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
		// This should not happen with well-formed structs, but handle it.
		fmt.Fprintf(os.Stderr, "Error: failed to marshal YAML output: %v\n", err)
		os.Exit(2)
	}

	fmt.Print(string(data))

	// Step 7: Exit with appropriate code.
	if len(specResults) > 0 || len(testResults) > 0 || len(codeResults) > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

// buildFrontmatterCache parses frontmatter for every discovered node
// and returns a map from file path to *Frontmatter. On parse failure,
// the entry is nil (not absent).
func buildFrontmatterCache(specNodes, testNodes []discovery.DiscoveredNode) map[string]*frontmatter.Frontmatter {
	cache := make(map[string]*frontmatter.Frontmatter)

	for _, node := range specNodes {
		fm, err := frontmatter.ParseFrontmatter(node.FilePath)
		if err != nil {
			// Store nil — parse failure is not an operational error.
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

// sortNodesByName sorts a slice of DiscoveredNode alphabetically by LogicalName.
func sortNodesByName(nodes []discovery.DiscoveredNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].LogicalName < nodes[j].LogicalName
	})
}

// collectSpecStaleness runs CheckSpecStaleness for each node and collects
// results into specStalenessEntry values. Only nodes with problems are included.
func collectSpecStaleness(nodes []discovery.DiscoveredNode, cache map[string]*frontmatter.Frontmatter) []specStalenessEntry {
	var entries []specStalenessEntry

	for _, node := range nodes {
		results := specstaleness.CheckSpecStaleness(node, cache)
		if len(results) == 0 {
			continue
		}

		// Collect all statuses for this node into a single entry.
		statuses := make([]string, 0, len(results))
		for _, r := range results {
			statuses = append(statuses, r.Status)
		}

		entries = append(entries, specStalenessEntry{
			Node:     node.LogicalName,
			Statuses: statuses,
		})
	}

	// Return empty slice (not nil) so YAML serializes as [].
	if entries == nil {
		entries = []specStalenessEntry{}
	}
	return entries
}

// collectCodeStaleness runs CheckCodeStaleness for each node and collects
// results into codeStalenessEntry values. Only files with problems are included.
func collectCodeStaleness(nodes []discovery.DiscoveredNode, cache map[string]*frontmatter.Frontmatter) []codeStalenessEntry {
	var entries []codeStalenessEntry

	for _, node := range nodes {
		results := codestaleness.CheckCodeStaleness(node, cache)
		for _, r := range results {
			entries = append(entries, codeStalenessEntry{
				Node:   r.Node,
				File:   r.File,
				Status: r.Status,
			})
		}
	}

	// Return empty slice (not nil) so YAML serializes as [].
	if entries == nil {
		entries = []codeStalenessEntry{}
	}
	return entries
}
