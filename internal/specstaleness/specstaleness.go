// code-from-spec: ROOT/tech_design/internal/spec_staleness@v13
package specstaleness

// Package specstaleness verifies spec staleness for a single node.
// The caller invokes CheckSpecStaleness once per discovered node and
// collects the results.
//
// Spec nodes (ROOT/ prefix) and test nodes (TEST/ prefix) follow
// different algorithms for the parent/subject check. All other checks
// are identical.

import (
	"strings"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/discovery"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/logicalnames"
)

// StalenessResult represents a single staleness problem detected for
// a node. A node may produce multiple results (e.g., wrong_name and
// parent_changed simultaneously). The File field is the node's file
// path; Status is one of the status strings defined in
// ROOT/domain/output.
type StalenessResult struct {
	Node   string
	File   string
	Status string
}

// CheckSpecStaleness checks one node for spec staleness. Returns an
// empty slice if the node is not stale. Returns one StalenessResult
// per problem found — a node may have multiple problems (e.g.,
// wrong name, parent changed, and dependency changed simultaneously).
//
// The cache maps file paths to parsed frontmatters, populated by the
// caller before invoking this function. Every discovered node has an
// entry in the cache: a valid *Frontmatter on success, or nil if
// frontmatter parsing failed. If a file path has no entry in the
// cache, the file does not exist.
//
// Spec nodes (ROOT/ prefix) and test nodes (TEST/ prefix) follow
// different algorithms for the parent/subject check.
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
// Steps 1-2 are blocking — if they fail, return immediately with a
// single result. From step 3 onward, all problems are collected.
func checkSpecNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	// If not found or nil → return [invalid_frontmatter].
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
		// Non-root spec node is missing the required parent_version field.
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// From step 3 onward, collect all problems found rather than
	// returning immediately.
	var results []StalenessResult

	// Step 3: Use LogicalNamesMatch to compare the frontmatter Title
	// against the node's LogicalName. If it does not match or Title
	// is empty → collect wrong_name.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Parent check. Only applies when the node has a parent.
	if ok && hasParent {
		parentLN, parentOK := logicalnames.ParentLogicalName(node.LogicalName)
		if parentOK {
			parentPath, pathOK := logicalnames.PathFromLogicalName(parentLN)
			if !pathOK {
				// Cannot resolve the parent's file path.
				results = append(results, makeResult(node, "invalid_parent"))
			} else {
				parentFM, parentExists := cache[parentPath]
				if !parentExists || parentFM == nil {
					// Parent frontmatter not found or failed to parse.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if parentFM.Version == nil {
					// Parent has no version field — treat as invalid.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if *fm.ParentVersion != *parentFM.Version {
					// Declared parent_version does not match the parent's
					// current version.
					results = append(results, makeResult(node, "parent_changed"))
				}
			}
		}
	}

	// Step 5: Dependency check. For each depends_on entry, resolve
	// the dependency's file path and compare versions.
	results = append(results, checkDependencies(node, fm.DependsOn, cache)...)

	// Step 6: Return all collected results (empty slice if none).
	return results
}

// checkTestNode implements the test node staleness algorithm.
// Steps 1-2 are blocking — if they fail, return immediately with a
// single result. From step 3 onward, all problems are collected.
func checkTestNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	// If not found or nil → return [invalid_frontmatter].
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Step 2: Check required fields. Both version and subject_version
	// must be present for test nodes.
	if fm.Version == nil || fm.SubjectVersion == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// From step 3 onward, collect all problems found.
	var results []StalenessResult

	// Step 3: Name verification — same as spec nodes. Compare the
	// frontmatter Title against the node's LogicalName.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Subject check. Use ParentLogicalName to derive the
	// subject's logical name (for test nodes this returns the
	// corresponding ROOT node), then resolve its file path.
	subjectLN, subjectOK := logicalnames.ParentLogicalName(node.LogicalName)
	if subjectOK {
		subjectPath, pathOK := logicalnames.PathFromLogicalName(subjectLN)
		if !pathOK {
			// Cannot resolve the subject's file path.
			results = append(results, makeResult(node, "invalid_subject"))
		} else {
			subjectFM, subjectExists := cache[subjectPath]
			if !subjectExists || subjectFM == nil {
				// Subject frontmatter not found or failed to parse.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if subjectFM.Version == nil {
				// Subject has no version field — treat as invalid.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if *fm.SubjectVersion != *subjectFM.Version {
				// Declared subject_version does not match the subject's
				// current version.
				results = append(results, makeResult(node, "subject_changed"))
			}
		}
	}

	// Step 5: Dependency check — same algorithm as spec nodes.
	results = append(results, checkDependencies(node, fm.DependsOn, cache)...)

	// Step 6: Return all collected results (empty slice if none).
	return results
}

// checkDependencies checks each depends_on entry for staleness and
// returns one StalenessResult per problem found. This logic is shared
// between spec nodes and test nodes.
func checkDependencies(
	node discovery.DiscoveredNode,
	deps []frontmatter.DependsOn,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	var results []StalenessResult

	for _, dep := range deps {
		// Resolve the dependency's file path from its logical name
		// using PathFromLogicalName.
		depPath, pathOK := logicalnames.PathFromLogicalName(dep.Path)
		if !pathOK {
			// Cannot resolve dependency path — collect invalid_dependency.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		depFM, depExists := cache[depPath]
		if !depExists || depFM == nil {
			// Dependency frontmatter not found or failed to parse.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		if depFM.Version == nil {
			// Dependency has no version field — treat as invalid.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		// Compare the declared depends_on version against the
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
