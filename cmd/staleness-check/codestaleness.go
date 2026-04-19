// spec: ROOT/tech_design/code_staleness@v2
package main

import (
	"os"
	"strings"
)

// CheckCodeStaleness checks one discovered node for code staleness. It returns
// an empty slice if all files are up to date or the node has no `implements`.
// Returns one StalenessResult per problem found.
//
// The cache maps file paths (relative to code-from-spec/) to parsed
// frontmatters. A nil value means frontmatter parsing failed. A missing key
// means the file does not exist.
//
// Algorithm (from ROOT/tech_design/code_staleness):
//   1. Look up frontmatter in cache — missing or nil → unreadable_frontmatter (blocking)
//   2. Check Version is not nil — nil → no_version (blocking)
//   3. If Implements is empty → return empty slice (blocking)
//   4. For each file in Implements, check conditions in order and produce
//      at most one StalenessResult per file.
func CheckCodeStaleness(
	node DiscoveredNode,
	cache map[string]*Frontmatter,
) []StalenessResult {

	// --- Step 1: Look up the node's frontmatter in the cache. ---
	// Missing key means the file does not exist; nil value means parsing failed.
	// Either way, we cannot proceed — return unreadable_frontmatter immediately.
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []StalenessResult{{
			Node:   node.LogicalName,
			Status: "unreadable_frontmatter",
		}}
	}

	// --- Step 2: Check that Version is not nil. ---
	// Frontmatter.Version is *int. A nil pointer means the version field is
	// absent from the frontmatter YAML — return no_version immediately.
	if fm.Version == nil {
		return []StalenessResult{{
			Node:   node.LogicalName,
			Status: "no_version",
		}}
	}

	// --- Step 3: If Implements is empty, there are no generated files to check. ---
	// Return an empty (nil) slice — nothing to do for this node.
	if len(fm.Implements) == 0 {
		return nil
	}

	// --- Step 4: Check each file in Implements. ---
	// For each file, apply conditions in order. The first matching condition
	// determines the status. If none match, the file is up to date and is
	// omitted from results.
	var results []StalenessResult

	for _, filePath := range fm.Implements {
		// Check condition 4a: File does not exist.
		// Use os.Stat to determine existence. Any error (including permission
		// errors) is treated as "file does not exist" per the spec.
		_, statErr := os.Stat(filePath)
		if statErr != nil {
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				File:   filePath,
				Status: "missing",
			})
			continue
		}

		// Check condition 4b/4c: Parse the spec comment from the file.
		// ParseSpecComment returns (*SpecComment, error). We distinguish
		// between "no spec comment" and "malformed" based on the error
		// message content.
		sc, parseErr := ParseSpecComment(filePath)
		if parseErr != nil {
			errMsg := parseErr.Error()

			// Check condition 4b: No spec comment found in the file.
			if strings.Contains(errMsg, "no spec comment found") {
				results = append(results, StalenessResult{
					Node:   node.LogicalName,
					File:   filePath,
					Status: "no_spec_comment",
				})
				continue
			}

			// Check condition 4c: Malformed spec comment.
			if strings.Contains(errMsg, "malformed spec comment") {
				results = append(results, StalenessResult{
					Node:   node.LogicalName,
					File:   filePath,
					Status: "malformed_spec_comment",
				})
				continue
			}

			// Any other error (e.g., file read error after stat succeeded)
			// — treat as no_spec_comment since we cannot read the comment.
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				File:   filePath,
				Status: "no_spec_comment",
			})
			continue
		}

		// Check condition 4d: Spec comment references a different node.
		// Use LogicalNamesMatch from logicalnames.go which handles the
		// TEST/x vs TEST/x(default) equivalence.
		if !LogicalNamesMatch(sc.LogicalName, node.LogicalName) {
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				File:   filePath,
				Status: "wrong_node",
			})
			continue
		}

		// Check condition 4e: Version mismatch → stale.
		// Compare the node's version (*fm.Version, known non-nil from step 2)
		// against the spec comment's version.
		if *fm.Version != sc.Version {
			results = append(results, StalenessResult{
				Node:   node.LogicalName,
				File:   filePath,
				Status: "stale",
			})
			continue
		}

		// None of the above conditions matched — the file is up to date.
		// Omit from results (no append).
	}

	return results
}
