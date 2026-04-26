// code-from-spec: TEST/tech_design/internal/spec_staleness@v8
package specstaleness

import (
	"testing"

	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/discovery"
	"github.com/CodeFromSpec/tool-staleness-check/v2/internal/frontmatter"
)

// intPtr is a helper to create *int values for Frontmatter fields.
func intPtr(v int) *int {
	return &v
}

// --- Happy Path — Spec Nodes ---

func TestAllChecksPass_SpecNode(t *testing.T) {
	// Parent version matches, title matches — no staleness.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

func TestAllChecksPass_RootNode(t *testing.T) {
	// Root node has no parent — only version and title checked.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/_node.md": {
			Version: intPtr(7),
			Title:   "ROOT",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT",
		FilePath:    "code-from-spec/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

func TestAllChecksPass_SpecNodeWithDependencies(t *testing.T) {
	// Node with a dependency — all versions match.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/tech_design/main/_node.md": {
			Version:       intPtr(3),
			ParentVersion: intPtr(10),
			Title:         "ROOT/tech_design/main",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness", Version: 6},
			},
		},
		"code-from-spec/tech_design/_node.md": {
			Version: intPtr(10),
			Title:   "ROOT/tech_design",
		},
		"code-from-spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/tech_design/main",
		FilePath:    "code-from-spec/tech_design/main/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

// --- Happy Path — Test Nodes ---

func TestAllChecksPass_TestNode(t *testing.T) {
	// Test node with subject version matching — no staleness.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config",
		},
		"code-from-spec/domain/config/_node.md": {
			Version: intPtr(2),
			Title:   "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

func TestAllChecksPass_NamedTestNode(t *testing.T) {
	// Named test node — subject version matches.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/edge_cases.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config(edge_cases)",
		},
		"code-from-spec/domain/config/_node.md": {
			Version: intPtr(2),
			Title:   "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config(edge_cases)",
		FilePath:    "code-from-spec/domain/config/edge_cases.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

func TestWrongName_TESTCanonicalVsTESTDefault(t *testing.T) {
	// LogicalNamesMatch treats TEST/domain/config and
	// TEST/domain/config(default) as equal — no wrong_name.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config(default)",
		},
		"code-from-spec/domain/config/_node.md": {
			Version: intPtr(2),
			Title:   "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

// --- Blocking Steps (1-2) — Spec Nodes ---

func TestNodeNotInCache_SpecNode(t *testing.T) {
	// Empty cache — node not found.
	cache := map[string]*frontmatter.Frontmatter{}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestNodeNilInCache_SpecNode(t *testing.T) {
	// Cache entry is nil — frontmatter parsing failed.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": nil,
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestVersionMissing_SpecNode(t *testing.T) {
	// Version is nil — required field missing.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       nil,
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestParentVersionMissing_NonRoot(t *testing.T) {
	// Non-root node with nil ParentVersion — required field missing.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: nil,
			Title:         "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestParentVersionMissing_RootIsOk(t *testing.T) {
	// Root node does not require parent_version.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/_node.md": {
			Version:       intPtr(7),
			ParentVersion: nil,
			Title:         "ROOT",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT",
		FilePath:    "code-from-spec/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

// --- Blocking Steps (1-2) — Test Nodes ---

func TestNodeNotInCache_TestNode(t *testing.T) {
	// Empty cache — test node not found.
	cache := map[string]*frontmatter.Frontmatter{}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestVersionMissing_TestNode(t *testing.T) {
	// Test node with nil Version.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        nil,
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

func TestSubjectVersionMissing_TestNode(t *testing.T) {
	// Test node with nil SubjectVersion — required for test nodes.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: nil,
			Title:          "TEST/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

// --- Individual Statuses — Spec Nodes ---

func TestWrongName_TitleMismatch(t *testing.T) {
	// Title says "ROOT/domain/old_name" but node is "ROOT/domain/config".
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/old_name",
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "wrong_name" {
		t.Errorf("expected status wrong_name, got %s", results[0].Status)
	}
}

func TestWrongName_EmptyTitle(t *testing.T) {
	// Empty title should produce wrong_name.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "",
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "wrong_name") {
		t.Errorf("expected results to include wrong_name, got %v", results)
	}
}

func TestInvalidParent_NotInCache(t *testing.T) {
	// Parent path resolves but is not in cache.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		// code-from-spec/domain/_node.md deliberately absent
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_parent") {
		t.Errorf("expected results to include invalid_parent, got %v", results)
	}
}

func TestInvalidParent_NilInCache(t *testing.T) {
	// Parent path is in cache but value is nil.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/domain/_node.md": nil,
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_parent") {
		t.Errorf("expected results to include invalid_parent, got %v", results)
	}
}

func TestParentChanged(t *testing.T) {
	// ParentVersion is 5 but parent's actual version is 6.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "parent_changed" {
		t.Errorf("expected status parent_changed, got %s", results[0].Status)
	}
}

// --- Individual Statuses — Test Nodes ---

func TestInvalidSubject_NotInCache(t *testing.T) {
	// Subject (the _node.md for the test's directory) is not in cache.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config",
		},
		// code-from-spec/domain/config/_node.md deliberately absent
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_subject") {
		t.Errorf("expected results to include invalid_subject, got %v", results)
	}
}

func TestInvalidSubject_NilInCache(t *testing.T) {
	// Subject is in cache but value is nil.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config",
		},
		"code-from-spec/domain/config/_node.md": nil,
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_subject") {
		t.Errorf("expected results to include invalid_subject, got %v", results)
	}
}

func TestSubjectChanged(t *testing.T) {
	// SubjectVersion is 2 but subject's actual version is 3.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/config",
		},
		"code-from-spec/domain/config/_node.md": {
			Version: intPtr(3),
			Title:   "ROOT/domain/config",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "subject_changed" {
		t.Errorf("expected status subject_changed, got %s", results[0].Status)
	}
}

// --- Dependency Statuses (Both Node Types) ---

func TestInvalidDependency_PathCannotBeResolved(t *testing.T) {
	// Dependency path "INVALID/bad" cannot be resolved by PathFromLogicalName.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "INVALID/bad", Version: 1},
			},
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %v", results)
	}
}

func TestInvalidDependency_NotInCache(t *testing.T) {
	// Dependency path resolves but is not in cache.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness", Version: 6},
			},
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		// code-from-spec/domain/staleness/_node.md deliberately absent
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if !hasStatus(results, "invalid_dependency") {
		t.Errorf("expected results to include invalid_dependency, got %v", results)
	}
}

func TestDependencyChanged(t *testing.T) {
	// DependsOn version is 4 but dependency's actual version is 6.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness", Version: 4},
			},
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		"code-from-spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "dependency_changed" {
		t.Errorf("expected status dependency_changed, got %s", results[0].Status)
	}
}

func TestDependencyWithSubsectionQualifier(t *testing.T) {
	// Dependency path has a subsection qualifier "(interface)" that
	// should be stripped during resolution. Version matches — no staleness.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/config",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness(interface)", Version: 6},
			},
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(5),
			Title:   "ROOT/domain",
		},
		"code-from-spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 0 {
		t.Errorf("expected empty slice, got %v", results)
	}
}

// --- Accumulation ---

func TestMultipleProblems_SpecNode(t *testing.T) {
	// Three problems: wrong title, parent changed (5 vs 6),
	// dependency changed (4 vs 6).
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": {
			Version:       intPtr(2),
			ParentVersion: intPtr(5),
			Title:         "ROOT/domain/old_name",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness", Version: 4},
			},
		},
		"code-from-spec/domain/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain",
		},
		"code-from-spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if !hasStatus(results, "wrong_name") {
		t.Errorf("expected wrong_name in results, got %v", results)
	}
	if !hasStatus(results, "parent_changed") {
		t.Errorf("expected parent_changed in results, got %v", results)
	}
	if !hasStatus(results, "dependency_changed") {
		t.Errorf("expected dependency_changed in results, got %v", results)
	}
}

func TestMultipleProblems_TestNode(t *testing.T) {
	// Three problems: wrong title, subject changed (2 vs 3),
	// dependency changed (4 vs 6).
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/default.test.md": {
			Version:        intPtr(1),
			SubjectVersion: intPtr(2),
			Title:          "TEST/domain/wrong",
			DependsOn: []frontmatter.DependsOnEntry{
				{Path: "ROOT/domain/staleness", Version: 4},
			},
		},
		"code-from-spec/domain/config/_node.md": {
			Version: intPtr(3),
			Title:   "ROOT/domain/config",
		},
		"code-from-spec/domain/staleness/_node.md": {
			Version: intPtr(6),
			Title:   "ROOT/domain/staleness",
		},
	}

	node := discovery.DiscoveredNode{
		LogicalName: "TEST/domain/config",
		FilePath:    "code-from-spec/domain/config/default.test.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(results), results)
	}
	if !hasStatus(results, "wrong_name") {
		t.Errorf("expected wrong_name in results, got %v", results)
	}
	if !hasStatus(results, "subject_changed") {
		t.Errorf("expected subject_changed in results, got %v", results)
	}
	if !hasStatus(results, "dependency_changed") {
		t.Errorf("expected dependency_changed in results, got %v", results)
	}
}

func TestBlockingStepPreventsAccumulation(t *testing.T) {
	// Nil cache entry blocks at step 1 — only one result returned.
	cache := map[string]*frontmatter.Frontmatter{
		"code-from-spec/domain/config/_node.md": nil,
	}

	node := discovery.DiscoveredNode{
		LogicalName: "ROOT/domain/config",
		FilePath:    "code-from-spec/domain/config/_node.md",
	}

	results := CheckSpecStaleness(node, cache)
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d: %v", len(results), results)
	}
	if results[0].Status != "invalid_frontmatter" {
		t.Errorf("expected status invalid_frontmatter, got %s", results[0].Status)
	}
}

// hasStatus is a test helper that checks whether any result in the
// slice has the given status string.
func hasStatus(results []StalenessResult, status string) bool {
	for _, r := range results {
		if r.Status == status {
			return true
		}
	}
	return false
}
