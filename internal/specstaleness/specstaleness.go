// code-from-spec: ROOT/tech_design/internal/spec_staleness@v10
package specstaleness

import (
	"strings"

	"github.com/gsneto/tool-staleness-check/internal/discovery"
	"github.com/gsneto/tool-staleness-check/internal/frontmatter"
	"github.com/gsneto/tool-staleness-check/internal/logicalnames"
)

// StalenessResult represents a single staleness problem detected
// for a node. A node may produce multiple results (e.g., wrong_name
// and parent_changed simultaneously). The File field is the node's
// file path; Status is one of the status strings defined in
// ROOT/domain/output.
type StalenessResult struct {
	Node   string
	File   string
	Status string
}

// CheckSpecStaleness checks one node for spec staleness. Returns an
// empty slice if the node is not stale. Returns one StalenessResult
// per problem found.
//
// The cache maps file paths to parsed frontmatters, populated by the
// caller. A valid *Frontmatter on success, nil if parsing failed.
// If a file path has no entry in the cache, the file does not exist.
//
// Spec nodes (ROOT/ prefix) and test nodes (TEST/ prefix) follow
// different algorithms for parent/subject checks.
func CheckSpecStaleness(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Determine whether this is a test node or a spec node based
	// on the logical name prefix.
	if strings.HasPrefix(node.LogicalName, "TEST") {
		return checkTestNode(node, cache)
	}
	return checkSpecNode(node, cache)
}

// checkSpecNode implements the spec node staleness algorithm.
// Steps 1-2 are blocking (return immediately on failure).
// Steps 3-5 collect all problems found.
func checkSpecNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	// If not found or nil, return invalid_frontmatter.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Step 2: Check required fields. version must be present.
	// parent_version must be present for non-root nodes.
	if fm.Version == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}
	// Determine if this node has a parent (i.e., it is not ROOT).
	hasParent, ok := logicalnames.HasParent(node.LogicalName)
	if ok && hasParent && fm.ParentVersion == nil {
		// Non-root spec node missing parent_version.
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// From here on, collect all problems found.
	var results []StalenessResult

	// Step 3: Name verification. Compare the frontmatter Title
	// against the node's LogicalName using LogicalNamesMatch.
	// If Title is empty or does not match, collect wrong_name.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Parent check. Only if the node has a parent.
	if ok && hasParent {
		parentLN, parentOK := logicalnames.ParentLogicalName(node.LogicalName)
		if parentOK {
			parentPath, pathOK := logicalnames.PathFromLogicalName(parentLN)
			if !pathOK {
				// Cannot resolve parent's file path.
				results = append(results, makeResult(node, "invalid_parent"))
			} else {
				parentFM, parentExists := cache[parentPath]
				if !parentExists || parentFM == nil {
					// Parent frontmatter not found or unparseable.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if parentFM.Version == nil {
					// Parent has no version — treat as invalid.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if *fm.ParentVersion != *parentFM.Version {
					// parent_version mismatch.
					results = append(results, makeResult(node, "parent_changed"))
				}
			}
		}
	}

	// Step 5: Dependency check. For each depends_on entry, resolve
	// the dependency's file path and compare versions.
	results = append(results, checkDependencies(node, fm.DependsOn, cache)...)

	// Step 6: Return collected results (empty slice if none).
	return results
}

// checkTestNode implements the test node staleness algorithm.
// Steps 1-2 are blocking (return immediately on failure).
// Steps 3-5 collect all problems found.
func checkTestNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Step 2: Check required fields. version and subject_version
	// must be present for test nodes.
	if fm.Version == nil || fm.SubjectVersion == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	var results []StalenessResult

	// Step 3: Name verification — same as spec nodes.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Subject check. Use ParentLogicalName to derive the
	// subject's logical name (for test nodes this returns the
	// subject, i.e., the ROOT node in the same directory).
	subjectLN, subjectOK := logicalnames.ParentLogicalName(node.LogicalName)
	if subjectOK {
		subjectPath, pathOK := logicalnames.PathFromLogicalName(subjectLN)
		if !pathOK {
			// Cannot resolve subject's file path.
			results = append(results, makeResult(node, "invalid_subject"))
		} else {
			subjectFM, subjectExists := cache[subjectPath]
			if !subjectExists || subjectFM == nil {
				// Subject frontmatter not found or unparseable.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if subjectFM.Version == nil {
				// Subject has no version — treat as invalid.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if *fm.SubjectVersion != *subjectFM.Version {
				// subject_version mismatch.
				results = append(results, makeResult(node, "subject_changed"))
			}
		}
	}

	// Step 5: Dependency check — same as spec nodes.
	results = append(results, checkDependencies(node, fm.DependsOn, cache)...)

	return results
}

// checkDependencies checks each depends_on entry for staleness.
// Returns one StalenessResult per problem found.
func checkDependencies(
	node discovery.DiscoveredNode,
	deps []frontmatter.DependsOn,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	var results []StalenessResult

	for _, dep := range deps {
		// Resolve the dependency's file path from its logical name.
		depPath, pathOK := logicalnames.PathFromLogicalName(dep.Path)
		if !pathOK {
			// Cannot resolve dependency path — invalid dependency.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		depFM, depExists := cache[depPath]
		if !depExists || depFM == nil {
			// Dependency frontmatter not found or unparseable.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		if depFM.Version == nil {
			// Dependency has no version — treat as invalid.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		// Compare the declared dependency version against the
		// dependency's current version.
		if dep.Version != *depFM.Version {
			results = append(results, makeResult(node, "dependency_changed"))
		}
	}

	return results
}

// makeResult is a convenience helper that builds a StalenessResult
// from a discovered node and a status string.
func makeResult(node discovery.DiscoveredNode, status string) StalenessResult {
	return StalenessResult{
		Node:   node.LogicalName,
		File:   node.FilePath,
		Status: status,
	}
}
