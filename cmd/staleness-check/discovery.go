// spec: ROOT/tech_design/discovery@v7
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoveredNode represents a single discovered spec node, test node,
// or external dependency with its logical name and file path.
type DiscoveredNode struct {
	LogicalName string
	FilePath    string
}

// DiscoverNodes walks the filesystem to discover all spec nodes, test nodes,
// and external dependencies. It returns three sorted slices (sorted
// alphabetically by LogicalName) and an error for operational failures.
//
// Discovery rules (from ROOT/tech_design/discovery):
//   - Walk code-from-spec/spec/ recursively: _node.md -> spec node, *.test.md -> test node
//   - Walk code-from-spec/external/ one level deep: _external.md -> external dependency
//
// Logical names are derived via LogicalNameFromPath (ROOT/tech_design/logical_names).
func DiscoverNodes() (
	specNodes []DiscoveredNode,
	testNodes []DiscoveredNode,
	externalDeps []DiscoveredNode,
	err error,
) {
	// Discover spec nodes and test nodes by walking code-from-spec/spec/.
	specNodes, testNodes, err = discoverSpecAndTestNodes()
	if err != nil {
		return nil, nil, nil, err
	}

	// Discover external dependencies by walking code-from-spec/external/
	// one level deep. The directory is optional — no error if missing.
	externalDeps, err = discoverExternalDeps()
	if err != nil {
		return nil, nil, nil, err
	}

	// Sort all three lists alphabetically by LogicalName.
	sort.Slice(specNodes, func(i, j int) bool {
		return specNodes[i].LogicalName < specNodes[j].LogicalName
	})
	sort.Slice(testNodes, func(i, j int) bool {
		return testNodes[i].LogicalName < testNodes[j].LogicalName
	})
	sort.Slice(externalDeps, func(i, j int) bool {
		return externalDeps[i].LogicalName < externalDeps[j].LogicalName
	})

	return specNodes, testNodes, externalDeps, nil
}

// discoverSpecAndTestNodes walks code-from-spec/spec/ recursively, collecting
// _node.md files as spec nodes and *.test.md files as test nodes.
func discoverSpecAndTestNodes() ([]DiscoveredNode, []DiscoveredNode, error) {
	var specNodes []DiscoveredNode
	var testNodes []DiscoveredNode

	// The code-from-spec/spec/ directory must exist — it's an operational
	// error if missing.
	specDir := "code-from-spec/spec"
	if _, statErr := os.Stat(specDir); statErr != nil {
		return nil, nil, fmt.Errorf("code-from-spec/spec/ directory not found: %w", statErr)
	}

	walkErr := filepath.Walk(specDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories themselves; we only care about files.
		if info.IsDir() {
			return nil
		}

		// Normalize path separators to forward slashes for consistent
		// logical name derivation across platforms.
		normalizedPath := filepath.ToSlash(path)
		name := info.Name()

		if name == "_node.md" {
			// Spec node: derive logical name from path using the
			// centralized LogicalNameFromPath function.
			logicalName, ok := LogicalNameFromPath(normalizedPath)
			if !ok {
				return nil
			}
			specNodes = append(specNodes, DiscoveredNode{
				LogicalName: logicalName,
				FilePath:    normalizedPath,
			})
		} else if strings.HasSuffix(name, ".test.md") {
			// Test node: derive logical name from path using the
			// centralized LogicalNameFromPath function.
			logicalName, ok := LogicalNameFromPath(normalizedPath)
			if !ok {
				return nil
			}
			testNodes = append(testNodes, DiscoveredNode{
				LogicalName: logicalName,
				FilePath:    normalizedPath,
			})
		}

		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("error walking code-from-spec/spec/ directory: %w", walkErr)
	}

	return specNodes, testNodes, nil
}

// discoverExternalDeps walks code-from-spec/external/ one level deep, collecting
// _external.md files as external dependencies.
//
// The code-from-spec/external/ directory is optional per
// ROOT/domain/external_dependencies: "The external/ directory itself is
// optional — a project may have no external dependencies." If it doesn't
// exist, we return an empty slice with no error.
func discoverExternalDeps() ([]DiscoveredNode, error) {
	var deps []DiscoveredNode

	externalDir := "code-from-spec/external"

	// Check if code-from-spec/external/ exists. If not, return empty — not
	// an error.
	info, statErr := os.Stat(externalDir)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, nil
		}
		return nil, fmt.Errorf("error accessing code-from-spec/external/ directory: %w", statErr)
	}
	if !info.IsDir() {
		return nil, nil
	}

	// Read the entries directly under code-from-spec/external/ (one level
	// deep).
	entries, readErr := os.ReadDir(externalDir)
	if readErr != nil {
		return nil, fmt.Errorf("error reading code-from-spec/external/ directory: %w", readErr)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check for _external.md inside this directory.
		externalFile := filepath.ToSlash(
			filepath.Join(externalDir, entry.Name(), "_external.md"),
		)
		if _, statErr := os.Stat(externalFile); statErr != nil {
			// No _external.md in this directory — skip it silently.
			continue
		}

		// Derive logical name using the centralized LogicalNameFromPath
		// function.
		logicalName, ok := LogicalNameFromPath(externalFile)
		if !ok {
			continue
		}
		deps = append(deps, DiscoveredNode{
			LogicalName: logicalName,
			FilePath:    externalFile,
		})
	}

	return deps, nil
}
