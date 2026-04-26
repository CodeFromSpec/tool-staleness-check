// code-from-spec: ROOT/tech_design/internal/code_staleness@v15

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
	// This is a blocking condition — return immediately.
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
	// A frontmatter without a version cannot be compared.
	// This is a blocking condition — return immediately.
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
	// check — this node does not generate any files.
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
// Checks are applied in order — each is a prerequisite for the next.
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

	// Condition 2-3: Parse the spec comment from the file.
	// ParseSpecComment reads line by line and stops as soon as
	// the pattern is found (per tech_design constraint: never the
	// full file). Errors are distinguished using sentinel values:
	//   errors.Is(err, speccomment.ErrNoSpecComment) → no_spec_comment
	//   errors.Is(err, speccomment.ErrMalformed)     → malformed_spec_comment
	//   other non-nil error                           → I/O failure (treat as malformed)
	sc, err := speccomment.ParseSpecComment(filePath)
	if err != nil {
		// Check the more specific sentinel first: no spec comment at all.
		if errors.Is(err, speccomment.ErrNoSpecComment) {
			return specstaleness.StalenessResult{
				Node:   logicalName,
				File:   filePath,
				Status: "no_spec_comment",
			}, true
		}

		// Malformed spec comment — the comment was found but
		// could not be parsed into a valid logical name and version.
		// This also covers unexpected I/O errors, which are treated
		// as malformed since the file exists (Stat passed above).
		if errors.Is(err, speccomment.ErrMalformed) {
			return specstaleness.StalenessResult{
				Node:   logicalName,
				File:   filePath,
				Status: "malformed_spec_comment",
			}, true
		}

		// Any other error (e.g., I/O failure after Stat succeeded)
		// is also treated as malformed — we cannot read the comment.
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "malformed_spec_comment",
		}, true
	}

	// Condition 4: The spec comment references a different node.
	// Use LogicalNamesMatch which handles qualifier equivalence
	// (e.g., ROOT/x(qualifier) matches ROOT/x, TEST/x matches
	// TEST/x(default)).
	if !logicalnames.LogicalNamesMatch(sc.LogicalName, logicalName) {
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "wrong_node",
		}, true
	}

	// Condition 5: Version mismatch — the file is stale.
	// The node version must exactly equal the version in the spec
	// comment for the file to be considered up to date.
	if nodeVersion != sc.Version {
		return specstaleness.StalenessResult{
			Node:   logicalName,
			File:   filePath,
			Status: "stale",
		}, true
	}

	// None of the above conditions matched — file is up to date,
	// omit from results.
	return specstaleness.StalenessResult{}, false
}
