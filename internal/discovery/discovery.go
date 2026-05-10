// code-from-spec: ROOT/tech_design/internal/discovery@v17
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/logicalnames"
)

// DiscoveredNode represents a single discovered spec or test node.
// LogicalName is the canonical identifier (e.g., "ROOT/x/y" or "TEST/x/y").
// FilePath is the path to the node file relative to the project root,
// using forward slashes (e.g., "code-from-spec/x/y/_node.md").
type DiscoveredNode struct {
	LogicalName string
	FilePath    string
}

// DiscoverNodes walks the code-from-spec/ directory recursively and returns
// all discovered spec nodes (_node.md files) and test nodes (*.test.md files).
//
// Discovery rules (from spec):
//   - Every _node.md file produces a spec node.
//   - Every *.test.md file produces a test node.
//   - Logical names are derived using logicalnames.LogicalNameFromPath.
//   - Paths passed to LogicalNameFromPath are relative to the project root.
//
// Both returned lists are sorted alphabetically by LogicalName.
// FilePath values are relative to the project root using forward slashes.
//
// Returns an error if:
//   - code-from-spec/ directory does not exist
//   - an error occurs while walking the directory
//   - no _node.md files are found (directory exists but is empty of spec nodes)
func DiscoverNodes() (specNodes []DiscoveredNode, testNodes []DiscoveredNode, err error) {
	const specDir = "code-from-spec"

	// Verify the spec directory exists before attempting to walk it.
	// This gives a clear, actionable error message if the directory is missing.
	if _, statErr := os.Stat(specDir); statErr != nil {
		return nil, nil, fmt.Errorf("code-from-spec/ directory not found: %w", statErr)
	}

	// Walk the entire spec directory tree.
	// filepath.Walk visits files and directories in lexical order, but we
	// still sort afterward to ensure consistent alphabetical ordering by
	// LogicalName (which may not match filesystem order).
	walkErr := filepath.Walk(specDir, func(path string, info os.FileInfo, walkInnerErr error) error {
		// Propagate any error encountered while accessing this path.
		if walkInnerErr != nil {
			return walkInnerErr
		}

		// Skip directories — we only care about files.
		if info.IsDir() {
			return nil
		}

		fileName := info.Name()

		// Normalize path separators to forward slashes for cross-platform
		// consistency. The spec requires FilePath to be relative to the
		// project root (e.g., "code-from-spec/domain/config/_node.md").
		normalizedPath := filepath.ToSlash(path)

		// Classify the file as a spec node, test node, or irrelevant.
		isSpecNode := fileName == "_node.md"
		isTestNode := !isSpecNode && strings.HasSuffix(fileName, ".test.md")

		// Skip files that match neither pattern.
		if !isSpecNode && !isTestNode {
			return nil
		}

		// Derive the logical name from the normalized file path.
		// If the path does not match any known pattern (unexpected structure),
		// skip it silently — the caller should not fail due to unknown files.
		logicalName, ok := logicalnames.LogicalNameFromPath(normalizedPath)
		if !ok {
			return nil
		}

		node := DiscoveredNode{
			LogicalName: logicalName,
			FilePath:    normalizedPath,
		}

		if isSpecNode {
			specNodes = append(specNodes, node)
		} else {
			testNodes = append(testNodes, node)
		}

		return nil
	})

	// If the walk itself encountered an error (e.g., permission denied on a
	// subdirectory), wrap it with a descriptive message and return it.
	if walkErr != nil {
		return nil, nil, fmt.Errorf("error walking code-from-spec/ directory: %w", walkErr)
	}

	// Per spec: if code-from-spec/ contains no _node.md files, return an error.
	// This catches cases where the directory exists but is effectively empty
	// of spec content.
	if len(specNodes) == 0 {
		return nil, nil, fmt.Errorf("code-from-spec/ directory not found: no _node.md files found")
	}

	// Sort both lists alphabetically by LogicalName as required by the spec.
	sort.Slice(specNodes, func(i, j int) bool {
		return specNodes[i].LogicalName < specNodes[j].LogicalName
	})
	sort.Slice(testNodes, func(i, j int) bool {
		return testNodes[i].LogicalName < testNodes[j].LogicalName
	})

	return specNodes, testNodes, nil
}
