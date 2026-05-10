// code-from-spec: ROOT/tech_design/internal/code_staleness@v16

// Package codestaleness verifies code staleness for a single node.
// The caller invokes CheckCodeStaleness once per discovered node
// and collects the results.
package codestaleness

import (
	"errors"
	"os"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/discovery"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/frontmatter"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/logicalnames"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/speccomment"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/specstaleness"
)

// CheckCodeStaleness checks one node for code staleness.
//
// It returns an empty slice if all implemented files are up to date
// or if the node has no implements list. It returns one
// specstaleness.StalenessResult per problem found.
//
// The cache maps file paths to parsed frontmatters, populated by the
// caller before invoking this function. Every discovered node has an
// entry in the cache: a valid *frontmatter.Frontmatter on success,
// or nil if frontmatter parsing failed. If a file path has no entry
// in the cache, the file does not exist.
func CheckCodeStaleness(
	node discovery.DiscoveredNode,
	cache map[string]*frontmatter.Frontmatter,
) []specstaleness.StalenessResult {

	// -------------------------------------------------------
	// Step 1: Look up the node's frontmatter in the cache.
	// If not found or nil, the frontmatter is unreadable.
	// This is a blocking condition — return immediately with
	// a single result and do not proceed to further checks.
	// -------------------------------------------------------
	fm, exists := cache[node.FilePath]
	if !exists || fm == nil {
		return []specstaleness.StalenessResult{
			{
				Node:   node.LogicalName,
				File:   "",
				Status: "unreadable_frontmatter",
			},
		}
	}

	// -------------------------------------------------------
	// Step 2: Check that Version is not nil.
	// A frontmatter without a version field cannot be compared
	// against spec comment versions. This is a blocking
	// condition — return immediately with a single result.
	// -------------------------------------------------------
	if fm.Version == nil {
		return []specstaleness.StalenessResult{
			{
				Node:   node.LogicalName,
				File:   "",
				Status: "no_version",
			},
		}
	}

	// -------------------------------------------------------
	// Step 3: If Implements is empty, there is nothing to
	// check — this node does not generate any source files.
	// Return an empty slice immediately.
	// -------------------------------------------------------
	if len(fm.Implements) == 0 {
		return nil
	}

	// -------------------------------------------------------
	// Step 4: For each file in Implements, check staleness.
	// Produce at most one StalenessResult per file, based on
	// the first matching condition (checks are sequential
	// prerequisites). Files that pass all checks are omitted.
	// -------------------------------------------------------
	var results []specstaleness.StalenessResult

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
//
// Checks are applied in order — each is a sequential prerequisite for
// the next, as defined by the spec algorithm (Step 4):
//   1. File does not exist              → "missing"
//   2. No spec comment found            → "no_spec_comment"
//   3. Spec comment is malformed        → "malformed_spec_comment"
//   4. Spec comment references wrong node → "wrong_node"
//   5. Version mismatch                 → "stale"
func checkImplementedFile(
	logicalName string,
	nodeVersion int,
	filePath string,
) (specstaleness.StalenessResult, bool) {

	// Condition 1: File does not exist.
	// Use os.Stat to check existence; treat IsNotExist as missing.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "missing",
		}, true
	}

	// Conditions 2 & 3: Parse the spec comment from the file.
	//
	// ParseSpecComment reads line by line and stops as soon as
	// the pattern is found (satisfying the constraint: read only
	// enough to find the spec comment, never the full file).
	//
	// Errors are distinguished using sentinel error values:
	//   errors.Is(err, speccomment.ErrNoSpecComment) → "no_spec_comment"
	//   errors.Is(err, speccomment.ErrMalformed)     → "malformed_spec_comment"
	//   other non-nil error (unexpected I/O failure)  → "malformed_spec_comment"
	//     (the file exists per Stat above, so an I/O failure here
	//      is treated as an inability to read the comment)
	sc, err := speccomment.ParseSpecComment(filePath)
	if err != nil {
		if errors.Is(err, speccomment.ErrNoSpecComment) {
			return specstaleness.StalenessResult{
				Node:   logicalName,
				File:   filePath,
				Status: "no_spec_comment",
			}, true
		}

		// ErrMalformed, or any other unexpected error — treat as malformed.
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "malformed_spec_comment",
		}, true
	}

	// Condition 4: The spec comment references a different node.
	// LogicalNamesMatch handles qualifier equivalences, e.g.:
	//   ROOT/x(qualifier) == ROOT/x
	//   TEST/x == TEST/x(default)
	if !logicalnames.LogicalNamesMatch(sc.LogicalName, logicalName) {
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "wrong_node",
		}, true
	}

	// Condition 5: Version mismatch — the file was generated from
	// an older (or different) version of the spec node.
	// node.version must exactly equal spec_comment.Version.
	if nodeVersion != sc.Version {
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "stale",
		}, true
	}

	// None of the above conditions matched — file is up to date.
	// Omit from results.
	return specstaleness.StalenessResult{}, false
}
