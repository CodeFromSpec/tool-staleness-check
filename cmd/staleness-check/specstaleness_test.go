// spec: TEST/tech_design/spec_staleness@v2
package main

import (
	"testing"
)

// intPtr is a helper that returns a pointer to the given int value.
// Frontmatter.Version and Frontmatter.ParentVersion are *int, so we need
// this to construct test data without temporary variables.
func intPtr(n int) *int {
	return &n
}

// hasStatus returns true if the results slice contains at least one
// StalenessResult with the given status string. Used for accumulation
// tests where order may vary.
func hasStatus(results []StalenessResult, status string) bool {
	for _, r := range results {
		if r.Status == status {
			return true
		}
	}
	return false
}

// --------------------------------------------------------------------------
// Happy Path
// --------------------------------------------------------------------------

// TestAllChecksPass_SpecNode verifies that a fully valid spec node with
// matching title, matching parent version, and no dependencies returns
// an empty result slice.
func TestAllChecksPass_SpecNode(t *testing.T) {
	// Cache: parent at spec/domain/_node.md (Version=5),
	// child at spec/domain/config/_node.md (Version=2, ParentVersion=5,
	// Title matches logical name).
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results, got %d: %+v", len(results), results)
	}
}

// TestAllChecksPass_TestNode verifies that a fully valid test node
// (default.test.md) with matching parent version returns an empty slice.
func TestAllChecksPass_TestNode(t *testing.T) {
	// Cache: parent spec node at Version=2, test node at Version=1
	// with ParentVersion=2 pointing to that parent.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/default.test.md": {
			Version:       intPtr(1),
			ParentVersion: intPtr(2),
			Title:         "TEST/domain/config",
		},
		"code-from-spec/spec/domain/config/_node.md": {
			Version: intPtr(2),
			Title:   "ROOT/domain/config",
		},
	}

	node := DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results, got %d: %+v", len(results), results)
	}
}

// TestAllChecksPass_RootNode verifies that the root node (ROOT) with
// no parent to check returns an empty slice.
func TestAllChecksPass_RootNode(t *testing.T) {
	// Cache: root node with Version=7, no parent needed.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/_node.md": {
			Version: intPtr(7),
			Title:   "ROOT",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT",
		FilePath:    "code-from-spec/spec/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results for root node, got %d: %+v", len(results), results)
	}
}

// TestAllChecksPass_WithDependencies verifies that a node with dependencies
// all at correct versions returns an empty slice.
func TestAllChecksPass_WithDependencies(t *testing.T) {
	// Cache: node with two dependencies (ROOT and EXTERNAL), both matching.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/tech_design/main/_node.md": {
			Version:       intPtr(3),
			ParentVersion: intPtr(10),
			Title:         "ROOT/tech_design/main",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 6},
				{Path: "EXTERNAL/api", Version: 2},
			},
		},
		"code-from-spec/spec/tech_design/_node.md": {
			Version: intPtr(10),
			Title:   "ROOT/tech_design",
		},
		"code-from-spec/spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
		"code-from-spec/external/api/_external.md": {
			Version: intPtr(2),
			Title:   "EXTERNAL/api",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/tech_design/main",
		FilePath:    "code-from-spec/spec/tech_design/main/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results with valid dependencies, got %d: %+v", len(results), results)
	}
}

// --------------------------------------------------------------------------
// Blocking Steps (1-2)
// --------------------------------------------------------------------------

// TestNodeNotInCache verifies that a node whose file path is not in the
// cache at all returns a single invalid_frontmatter result.
func TestNodeNotInCache(t *testing.T) {
	// Empty cache — the node's file path is not present.
	cache := map[string]*Frontmatter{}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %q", results[0].Status)
	}
}

// TestNodeInCacheWithNil verifies that a node whose cache entry is nil
// (frontmatter parse failed) returns a single invalid_frontmatter result.
func TestNodeInCacheWithNil(t *testing.T) {
	// Cache has the key but value is nil — parse failure.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": nil,
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %q", results[0].Status)
	}
}

// TestVersionMissing verifies that a node with nil Version pointer returns
// a single invalid_frontmatter result (blocking step 2).
func TestVersionMissing(t *testing.T) {
	// Version is nil — required field missing.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       nil,
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %q", results[0].Status)
	}
}

// TestParentVersionMissingOnNonRoot verifies that a non-root node with nil
// ParentVersion pointer returns a single invalid_frontmatter result.
func TestParentVersionMissingOnNonRoot(t *testing.T) {
	// ParentVersion is nil on a non-root node — required field missing.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: nil,
			Title:         "ROOT/domain/config",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %q", results[0].Status)
	}
}

// TestParentVersionMissingOnRootIsOk verifies that the root node (ROOT)
// does not require parent_version — nil ParentVersion is acceptable.
func TestParentVersionMissingOnRootIsOk(t *testing.T) {
	// Root node with ParentVersion=nil — this is fine for ROOT.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/_node.md": {
			Version:       intPtr(7),
			ParentVersion: nil,
			Title:         "ROOT",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT",
		FilePath:    "code-from-spec/spec/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results for root with nil ParentVersion, got %d: %+v", len(results), results)
	}
}

// --------------------------------------------------------------------------
// Individual Statuses
// --------------------------------------------------------------------------

// TestWrongName_TitleMismatch verifies that a title that does not match
// the logical name produces a wrong_name result.
func TestWrongName_TitleMismatch(t *testing.T) {
	// Title says "ROOT/domain/old_name" but logical name is "ROOT/domain/config".
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/old_name",
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "wrong_name" {
		t.Errorf("expected status wrong_name, got %q", results[0].Status)
	}
}

// TestWrongName_EmptyTitle verifies that an empty title produces a result
// that includes wrong_name.
func TestWrongName_EmptyTitle(t *testing.T) {
	// Title is empty string — should trigger wrong_name.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "",
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "wrong_name") {
		t.Errorf("expected results to include wrong_name, got %+v", results)
	}
}

// TestWrongName_TESTCanonicalVsTESTDefault verifies that LogicalNamesMatch
// treats TEST/domain/config and TEST/domain/config(default) as equal,
// so no wrong_name is produced.
func TestWrongName_TESTCanonicalVsTESTDefault(t *testing.T) {
	// Title uses the explicit form TEST/domain/config(default) but the
	// logical name is the canonical form TEST/domain/config. These should
	// match via LogicalNamesMatch.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/default.test.md": {
			Version:       intPtr(1),
			ParentVersion: intPtr(2),
			Title:         "TEST/domain/config(default)",
		},
		"code-from-spec/spec/domain/config/_node.md": {
			Version: intPtr(2),
			Title:   "ROOT/domain/config",
		},
	}

	node := DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 0 {
		t.Errorf("expected empty results (TEST canonical matches TEST(default)), got %d: %+v", len(results), results)
	}
}

// TestInvalidParent_NotInCache verifies that when the parent's file path
// is not in the cache, the result includes invalid_parent.
func TestInvalidParent_NotInCache(t *testing.T) {
	// Node's parent (spec/domain/_node.md) is not in the cache at all.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "invalid_parent") {
		t.Errorf("expected results to include invalid_parent, got %+v", results)
	}
}

// TestInvalidParent_NilInCache verifies that when the parent's cache entry
// is nil (parse failed), the result includes invalid_parent.
func TestInvalidParent_NilInCache(t *testing.T) {
	// Parent exists in cache but its value is nil — frontmatter parse failed.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/spec/domain/_node.md": nil,
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "invalid_parent") {
		t.Errorf("expected results to include invalid_parent, got %+v", results)
	}
}

// TestParentChanged verifies that when the node's parent_version does not
// match the parent's version, a parent_changed result is produced.
func TestParentChanged(t *testing.T) {
	// Node says ParentVersion=5, but parent's Version is 6.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "parent_changed" {
		t.Errorf("expected status parent_changed, got %q", results[0].Status)
	}
}

// TestInvalidDependency_PathCannotBeResolved verifies that a dependency
// with an unresolvable logical name (e.g., "INVALID/bad") produces
// invalid_dependency.
func TestInvalidDependency_PathCannotBeResolved(t *testing.T) {
	// Dependency path "INVALID/bad" does not match any known prefix,
	// so PathFromLogicalName returns false.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []DependsOn{
				{Path: "INVALID/bad", Version: 1},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %+v", results)
	}
}

// TestInvalidDependency_NotInCache verifies that a dependency whose
// resolved file path is not in the cache produces invalid_dependency.
func TestInvalidDependency_NotInCache(t *testing.T) {
	// Dependency "ROOT/domain/staleness" resolves to
	// "code-from-spec/spec/domain/staleness/_node.md" but that path is not in the cache.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 6},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %+v", results)
	}
}

// TestInvalidDependency_NilInCache verifies that a dependency whose cache
// entry is nil (parse failed) produces invalid_dependency.
func TestInvalidDependency_NilInCache(t *testing.T) {
	// Dependency exists in cache but value is nil.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 6},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		"code-from-spec/spec/domain/staleness/_node.md": nil,
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %+v", results)
	}
}

// TestDependencyChanged verifies that when a dependency's version in the
// cache differs from the depends_on version, dependency_changed is produced.
func TestDependencyChanged(t *testing.T) {
	// DependsOn says Version=4, but the dependency's actual Version is 6.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 4},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		"code-from-spec/spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	if results[0].Status != "dependency_changed" {
		t.Errorf("expected status dependency_changed, got %q", results[0].Status)
	}
}

// --------------------------------------------------------------------------
// Accumulation
// --------------------------------------------------------------------------

// TestMultipleProblemsCollected verifies that multiple independent problems
// (wrong_name, parent_changed, dependency_changed) are all collected in
// a single call.
func TestMultipleProblemsCollected(t *testing.T) {
	// Three problems:
	// 1. Title is "ROOT/domain/old_name" but logical name is "ROOT/domain/config" → wrong_name
	// 2. ParentVersion=5 but parent Version=6 → parent_changed
	// 3. DependsOn Version=4 but dependency Version=6 → dependency_changed
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/old_name",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 4},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain",
		},
		"code-from-spec/spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	// Expect exactly 3 results — order may vary, so check by status.
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %+v", len(results), results)
	}
	if !hasStatus(results, "wrong_name") {
		t.Errorf("expected results to include wrong_name, got %+v", results)
	}
	if !hasStatus(results, "parent_changed") {
		t.Errorf("expected results to include parent_changed, got %+v", results)
	}
	if !hasStatus(results, "dependency_changed") {
		t.Errorf("expected results to include dependency_changed, got %+v", results)
	}
}

// TestMultipleDependencyProblems verifies that problems from multiple
// dependencies are each reported independently.
func TestMultipleDependencyProblems(t *testing.T) {
	// Two dependency problems:
	// 1. First dependency changed (Version=4 vs actual 6) → dependency_changed
	// 2. Second dependency not in cache → invalid_dependency
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []DependsOn{
				{Path: "ROOT/domain/staleness", Version: 4},
				{Path: "ROOT/domain/output", Version: 3},
			},
		},
		"code-from-spec/spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		"code-from-spec/spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
		// NOTE: spec/domain/output/_node.md is intentionally absent from cache.
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	// Expect exactly 2 results — order may vary.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	if !hasStatus(results, "dependency_changed") {
		t.Errorf("expected results to include dependency_changed, got %+v", results)
	}
	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %+v", results)
	}
}

// TestBlockingStepPreventsAccumulation verifies that when a blocking step
// fails (nil frontmatter in cache), exactly one invalid_frontmatter result
// is returned and no further checks are performed.
func TestBlockingStepPreventsAccumulation(t *testing.T) {
	// Cache entry is nil — blocking step 1 fires immediately.
	cache := map[string]*Frontmatter{
		"code-from-spec/spec/domain/config/_node.md": nil,
	}

	node := DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)

	// Must be exactly 1 result — no further checks after blocking step.
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result (blocking), got %d: %+v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %q", results[0].Status)
	}
}
