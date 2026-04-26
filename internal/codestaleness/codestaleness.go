// code-from-spec: ROOT/tech_design/internal/code_staleness@v9

// Package codestaleness verifies code staleness for a single node.
// The caller invokes CheckCodeStaleness once per discovered node
// and collects the results.
package codestaleness

import (
	"os"

	"github.com/gsneto/tool-staleness-check/internal/discovery"
	"github.com/gsneto/tool-staleness-check/internal/frontmatter"
	"github.com/gsneto/tool-staleness-check/internal/logicalnames"
	"github.com/gsneto/tool-staleness-check/internal/output"
	"github.com/gsneto/tool-staleness-check/internal/speccomment"
)

// CheckCodeStaleness checks one node for code staleness.
//
// It returns an empty slice if all implemented files are up to date
// or if the node has no implements list. It returns one
// output.StalenessResult per problem found.
//
// The cache maps file paths to parsed frontmatters, populated by the
// caller before invoking this function. Every discovered node has an
// entry in the cache: a valid *frontmatter.Frontmatter on success,
// or nil if frontmatter parsing failed. If a file path has no entry
// in the cache, the file does not exist.
func CheckCodeStaleness(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []output.StalenessResult {

	// -------------------------------------------------------
	// Step 1: Look up the node's frontmatter in the cache.
	// If not found or nil, the frontmatter is unreadable.
	// -------------------------------------------------------
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []output.StalenessResult{
			{
				Node:   node.LogicalName,
				File:   "",
				Status: "unreadable_frontmatter",
			},
		}
	}

	// -------------------------------------------------------
	// Step 2: Check that Version is not nil.
	// A frontmatter without a version cannot be compared.
	// -------------------------------------------------------
	if fm.Version == nil {
		return []output.StalenessResult{
			{
				Node:   node.LogicalName,
				File:   "",
				Status: "no_version",
			},
		}
	}

	// -------------------------------------------------------
	// Step 3: If Implements is empty, there is nothing to
	// check — this node does not generate any files.
	// -------------------------------------------------------
	if len(fm.Implements) == 0 {
		return nil
	}

	// -------------------------------------------------------
	// Step 4: For each file in Implements, check staleness.
	// Produce at most one StalenessResult per file, based on
	// the first matching condition (checks are sequential
	// prerequisites).
	// -------------------------------------------------------
	var results []output.StalenessResult

	for _, filePath := range fm.Implements {
		result, hasIssue := checkImplementedFile(node.LogicalName, *fm.Version, filePath)
		if hasIssue {
			results = append(results, result)
		}
	}

	return results
}

// checkImplementedFile checks a single implemented file for staleness.
// Returns (result, true) if a problem was found, or (zero, false) if
// the file is up to date.
func checkImplementedFile(
	logicalName string,
	nodeVersion int,
	filePath string,
) (output.StalenessResult, bool) {

	// Condition 1: File does not exist.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return output.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "missing",
		}, true
	}

	// Condition 2-3: Parse the spec comment from the file.
	// ParseSpecComment returns:
	//   - (nil, nil)  → no spec comment found
	//   - (nil, err)  → malformed spec comment
	//   - (sc, nil)   → valid spec comment
	sc, err := speccomment.ParseSpecComment(filePath)
	if err != nil {
		// Malformed spec comment — the comment was found but
		// could not be parsed into a valid logical name and version.
		return output.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "malformed_spec_comment",
		}, true
	}
	if sc == nil {
		// No spec comment found in the file at all.
		return output.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "no_spec_comment",
		}, true
	}

	// Condition 4: The spec comment references a different node.
	// Use LogicalNamesMatch which handles qualifier equivalence
	// (e.g., ROOT/x(qualifier) matches ROOT/x).
	if !logicalnames.LogicalNamesMatch(sc.LogicalName, logicalName) {
		return output.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "wrong_node",
		}, true
	}

	// Condition 5: Version mismatch — the file is stale.
	if nodeVersion != sc.Version {
		return output.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "stale",
		}, true
	}

	// None of the above — file is up to date, omit from results.
	return output.StalenessResult{}, false
}
