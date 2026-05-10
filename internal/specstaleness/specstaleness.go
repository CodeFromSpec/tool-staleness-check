// code-from-spec: ROOT/tech_design/internal/spec_staleness@v14
package specstaleness

// Package specstaleness verifies spec staleness for a single node.
// The caller invokes CheckSpecStaleness once per discovered node and
// collects the results.
//
// Spec nodes (ROOT/ prefix) and test nodes (TEST/ prefix) follow
// different algorithms for the parent/subject check. All other checks
// are identical.
//
// Statuses produced:
//   - invalid_frontmatter: frontmatter missing or required fields absent
//   - wrong_name:          title does not match logical name
//   - invalid_parent:      parent file not found or unreadable
//   - parent_changed:      node.parent_version != parent.version
//   - invalid_subject:     subject file not found or unreadable (test nodes)
//   - subject_changed:     node.subject_version != subject.version (test nodes)
//   - invalid_dependency:  depends_on entry cannot be resolved or read
//   - dependency_changed:  depends_on[].version != dependency.version

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
// per problem found — a node may have multiple problems (e.g., wrong
// name, parent changed, and dependency changed simultaneously).
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
	// Determine whether this is a test node or a spec node based on
	// the logical name prefix. All TEST/* nodes use the test algorithm;
	// all ROOT/* nodes use the spec algorithm.
	if strings.HasPrefix(node.LogicalName, "TEST") {
		return checkTestNode(node, cache)
	}
	return checkSpecNode(node, cache)
}

// checkSpecNode implements the spec node staleness algorithm described
// in ROOT/tech_design/internal/spec_staleness.
//
// Steps 1–2 are blocking: if they fail, return immediately with a
// single invalid_frontmatter result. From step 3 onward, all problems
// are collected and returned together.
func checkSpecNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	// No entry or nil entry means frontmatter could not be parsed.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Step 2: Validate required fields.
	//   - version must always be present.
	//   - parent_version must be present for non-root spec nodes.
	if fm.Version == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Determine whether the node has a parent using HasParent.
	// HasParent also tells us whether the logical name is valid (ok).
	hasParent, ok := logicalnames.HasParent(node.LogicalName)
	if ok && hasParent && fm.ParentVersion == nil {
		// Non-root spec node is missing the required parent_version field.
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Steps 3–5 collect all problems rather than returning early.
	var results []StalenessResult

	// Step 3: Name verification.
	// Use LogicalNamesMatch to compare the frontmatter Title against
	// the node's LogicalName. An empty title always fails.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Parent check — only for nodes that have a parent.
	if ok && hasParent {
		parentLN, parentOK := logicalnames.ParentLogicalName(node.LogicalName)
		if parentOK {
			parentPath, pathOK := logicalnames.PathFromLogicalName(parentLN)
			if !pathOK {
				// The parent logical name cannot be resolved to a file path.
				results = append(results, makeResult(node, "invalid_parent"))
			} else {
				parentFM, parentExists := cache[parentPath]
				if !parentExists || parentFM == nil {
					// Parent file does not exist or frontmatter failed to parse.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if parentFM.Version == nil {
					// Parent has no version field — treat as invalid.
					results = append(results, makeResult(node, "invalid_parent"))
				} else if *fm.ParentVersion != *parentFM.Version {
					// The declared parent_version does not match the parent's
					// current version — the parent has changed.
					results = append(results, makeResult(node, "parent_changed"))
				}
			}
		}
	}

	// Step 5: Dependency check — shared with test nodes.
	results = append(results, checkDependencies(node, fm.DependsOn, cache)...)

	// Step 6: Return all collected results (empty slice if none).
	return results
}

// checkTestNode implements the test node staleness algorithm described
// in ROOT/tech_design/internal/spec_staleness.
//
// Steps 1–2 are blocking: if they fail, return immediately with a
// single invalid_frontmatter result. From step 3 onward, all problems
// are collected and returned together.
func checkTestNode(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	// Step 1: Look up the node's frontmatter in the cache.
	// No entry or nil entry means frontmatter could not be parsed.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Step 2: Validate required fields.
	// Both version and subject_version must be present for test nodes.
	if fm.Version == nil || fm.SubjectVersion == nil {
		return []StalenessResult{makeResult(node, "invalid_frontmatter")}
	}

	// Steps 3–5 collect all problems rather than returning early.
	var results []StalenessResult

	// Step 3: Name verification — identical to spec nodes.
	// Use LogicalNamesMatch to compare the frontmatter Title against
	// the node's LogicalName. An empty title always fails.
	if fm.Title == "" || !logicalnames.LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, makeResult(node, "wrong_name"))
	}

	// Step 4: Subject check.
	// Use ParentLogicalName to derive the subject's logical name.
	// For test nodes, ParentLogicalName returns the corresponding ROOT node
	// (e.g., TEST/x → ROOT/x, TEST/x(name) → ROOT/x).
	subjectLN, subjectOK := logicalnames.ParentLogicalName(node.LogicalName)
	if subjectOK {
		subjectPath, pathOK := logicalnames.PathFromLogicalName(subjectLN)
		if !pathOK {
			// The subject logical name cannot be resolved to a file path.
			results = append(results, makeResult(node, "invalid_subject"))
		} else {
			subjectFM, subjectExists := cache[subjectPath]
			if !subjectExists || subjectFM == nil {
				// Subject file does not exist or frontmatter failed to parse.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if subjectFM.Version == nil {
				// Subject has no version field — treat as invalid.
				results = append(results, makeResult(node, "invalid_subject"))
			} else if *fm.SubjectVersion != *subjectFM.Version {
				// The declared subject_version does not match the subject's
				// current version — the subject has changed.
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
// between spec nodes and test nodes (algorithm step 5 in both cases).
//
// For each dependency:
//   - Resolve its logical name to a file path using PathFromLogicalName.
//   - If unresolvable or absent/nil in the cache → invalid_dependency.
//   - If the declared version does not match the current version → dependency_changed.
func checkDependencies(
	node discovery.DiscoveredNode,
	deps []frontmatter.DependsOn,
	cache map[string]*frontmatter.Frontmatter,
) []StalenessResult {
	var results []StalenessResult

	for _, dep := range deps {
		// Resolve the dependency's file path from its logical name.
		// PathFromLogicalName returns ("", false) for unrecognised patterns.
		depPath, pathOK := logicalnames.PathFromLogicalName(dep.Path)
		if !pathOK {
			// The depends_on entry contains an unresolvable logical name.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		depFM, depExists := cache[depPath]
		if !depExists || depFM == nil {
			// Dependency file does not exist or frontmatter failed to parse.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		if depFM.Version == nil {
			// Dependency has no version field — treat as invalid.
			results = append(results, makeResult(node, "invalid_dependency"))
			continue
		}

		// Compare the declared depends_on version against the dependency's
		// current version. A mismatch means the dependency has changed.
		if dep.Version != *depFM.Version {
			results = append(results, makeResult(node, "dependency_changed"))
		}
	}

	return results
}

// makeResult is a convenience helper that constructs a StalenessResult
// from a discovered node and a status string.
func makeResult(node discovery.DiscoveredNode, status string) StalenessResult {
	return StalenessResult{
		Node:   node.LogicalName,
		File:   node.FilePath,
		Status: status,
	}
}
