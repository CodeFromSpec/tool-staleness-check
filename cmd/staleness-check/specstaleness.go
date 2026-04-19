// spec: ROOT/tech_design/spec_staleness@v4
package main

// StalenessResult represents a single staleness problem detected for a node.
// A node may produce zero or more results — one per problem found. The Node
// field holds the logical name, File is empty for spec/test staleness (used
// only by code staleness), and Status is one of the defined status strings
// from ROOT/domain/output.
type StalenessResult struct {
	Node   string
	File   string
	Status string
}

// CheckSpecStaleness checks one discovered node for spec staleness. It returns
// an empty slice if the node has no problems, or one StalenessResult per
// problem found. A node may have multiple problems (e.g., wrong_name,
// parent_changed, and dependency_changed simultaneously).
//
// The cache maps file paths (relative to code-from-spec/) to parsed
// frontmatters. A nil value means frontmatter parsing failed. A missing key
// means the file does not exist.
//
// Algorithm steps (from ROOT/tech_design/spec_staleness):
//   1. Look up frontmatter in cache — if missing or nil → invalid_frontmatter (blocking)
//   2. Check required fields — version must be present; parent_version for non-root (blocking)
//   3. Name verification — title must match logical name
//   4. Parent check — parent version must match
//   5. Dependency check — each dependency version must match
//   6. Return collected results
func CheckSpecStaleness(
	node DiscoveredNode,
	cache map[string]*Frontmatter,
) []StalenessResult {

	// --- Step 1: Look up the node's frontmatter in the cache. ---
	// If the key is missing (file does not exist) or the value is nil
	// (frontmatter parsing failed), return immediately with invalid_frontmatter.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{{
			Node:   node.LogicalName,
			Status: "invalid_frontmatter",
		}}
	}

	// --- Step 2: Validate required fields. ---
	// version must always be present (pointer must be non-nil).
	// parent_version must be present for non-root nodes (nodes that have a parent).
	if fm.Version == nil {
		// Version is required for all nodes — missing means invalid frontmatter.
		return []StalenessResult{{
			Node:   node.LogicalName,
			Status: "invalid_frontmatter",
		}}
	}

	// Determine whether this node has a parent using HasParent from logicalnames.go.
	hasParent, ok := HasParent(node.LogicalName)
	if ok && hasParent && fm.ParentVersion == nil {
		// Non-root node must declare parent_version — missing means invalid frontmatter.
		return []StalenessResult{{
			Node:   node.LogicalName,
			Status: "invalid_frontmatter",
		}}
	}

	// --- Steps 3-6: Collect all problems found. ---
	var results []StalenessResult

	// --- Step 3: Name verification. ---
	// Compare the frontmatter Title against the node's LogicalName using
	// LogicalNamesMatch from logicalnames.go. If the title is empty or does
	// not match, collect wrong_name.
	if fm.Title == "" || !LogicalNamesMatch(fm.Title, node.LogicalName) {
		results = append(results, StalenessResult{
			Node:   node.LogicalName,
			Status: "wrong_name",
		})
	}

	// --- Step 4: Parent check. ---
	// Only performed if the node has a parent (determined above).
	if ok && hasParent {
		// Derive the parent's logical name, then resolve it to a file path.
		parentLN, parentLNOk := ParentLogicalName(node.LogicalName)
		if !parentLNOk {
			// Cannot derive parent logical name — should not happen if
			// HasParent returned true, but handle defensively.
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				Status: "invalid_parent",
			})
		} else {
			// Resolve parent logical name to file path.
			parentPath, parentPathOk := PathFromLogicalName(parentLN)
			if !parentPathOk {
				// Cannot resolve parent logical name to a file path.
				results = append(results, StalenessResult{
					Node:   node.LogicalName,
					Status: "invalid_parent",
				})
			} else {
				// Look up parent's frontmatter in the cache.
				parentFM, parentExists := cache[parentPath]
				if !parentExists || parentFM == nil {
					// Parent file does not exist or its frontmatter is invalid.
					results = append(results, StalenessResult{
						Node:   node.LogicalName,
						Status: "invalid_parent",
					})
				} else {
					// Compare: node.parent_version != parent.version → parent_changed.
					// Both ParentVersion and parent's Version are known non-nil at this point
					// (ParentVersion checked in step 2, parent's Version checked here).
					if parentFM.Version == nil {
						// Parent has no version — treat as invalid parent.
						results = append(results, StalenessResult{
							Node:   node.LogicalName,
							Status: "invalid_parent",
						})
					} else if *fm.ParentVersion != *parentFM.Version {
						results = append(results, StalenessResult{
							Node:   node.LogicalName,
							Status: "parent_changed",
						})
					}
				}
			}
		}
	}

	// --- Step 5: Dependency check. ---
	// For each entry in the frontmatter's depends_on list, resolve the
	// dependency's logical name to a file path, look it up in the cache,
	// and compare versions.
	for _, dep := range fm.DependsOn {
		// Resolve the dependency's logical name (dep.Path) to a file path.
		depPath, depPathOk := PathFromLogicalName(dep.Path)
		if !depPathOk {
			// Cannot resolve dependency logical name to a file path.
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				Status: "invalid_dependency",
			})
			continue
		}

		// Look up the dependency's frontmatter in the cache.
		depFM, depExists := cache[depPath]
		if !depExists || depFM == nil {
			// Dependency file does not exist or its frontmatter is invalid.
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				Status: "invalid_dependency",
			})
			continue
		}

		// Compare: depends_on.version != dependency.version → dependency_changed.
		if depFM.Version == nil {
			// Dependency has no version field — treat as invalid dependency.
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				Status: "invalid_dependency",
			})
			continue
		}

		if dep.Version != *depFM.Version {
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				Status: "dependency_changed",
			})
		}
	}

	// --- Step 6: Return all collected results. ---
	// Empty slice (nil) if no problems were found.
	return results
}
