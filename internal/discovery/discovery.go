// code-from-spec: ROOT/tech_design/internal/discovery@v13
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gsneto/tool-staleness-check/internal/logicalnames"
)

// DiscoveredNode represents a single discovered spec or test node.
type DiscoveredNode struct {
	LogicalName string
	FilePath    string
}

// DiscoverNodes walks the code-from-spec/ directory recursively and returns
// all discovered spec nodes (_node.md files) and test nodes (*.test.md files).
// Both lists are sorted alphabetically by LogicalName.
// FilePath values use forward slashes and are relative to the project root.
//
// Returns an error if:
//   - code-from-spec/ directory does not exist
//   - an error occurs while walking the directory
//   - no _node.md files are found
func DiscoverNodes() (specNodes []DiscoveredNode, testNodes []DiscoveredNode, err error) {
	const specDir = "code-from-spec"

	// Check that the spec directory exists before walking.
	if _, statErr := os.Stat(specDir); statErr != nil {
		return nil, nil, fmt.Errorf("code-from-spec/ directory not found: %w", statErr)
	}

	// Walk the spec directory to find all _node.md and *.test.md files.
	walkErr := filepath.Walk(specDir, func(path string, info os.FileInfo, walkInnerErr error) error {
		if walkInnerErr != nil {
			return walkInnerErr
		}

		// Skip directories — we only care about files.
		if info.IsDir() {
			return nil
		}

		fileName := info.Name()

		// Normalize path separators to forward slashes for consistency
		// across platforms. Paths are stored relative to the project root.
		normalizedPath := filepath.ToSlash(path)

		// Determine if this file is a spec node or a test node.
		isSpecNode := fileName == "_node.md"
		isTestNode := !isSpecNode && strings.HasSuffix(fileName, ".test.md")

		if !isSpecNode && !isTestNode {
			return nil
		}

		// Derive the logical name from the file path.
		logicalName, ok := logicalnames.LogicalNameFromPath(normalizedPath)
		if !ok {
			// If the path doesn't match any known pattern, skip it silently.
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

	if walkErr != nil {
		return nil, nil, fmt.Errorf("error walking code-from-spec/ directory: %w", walkErr)
	}

	// It is an error if no spec nodes were found at all.
	if len(specNodes) == 0 {
		return nil, nil, fmt.Errorf("code-from-spec/ directory not found: no _node.md files found")
	}

	// Sort both lists alphabetically by LogicalName.
	sort.Slice(specNodes, func(i, j int) bool {
		return specNodes[i].LogicalName < specNodes[j].LogicalName
	})
	sort.Slice(testNodes, func(i, j int) bool {
		return testNodes[i].LogicalName < testNodes[j].LogicalName
	})

	return specNodes, testNodes, nil
}
