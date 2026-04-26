---
version: 14
parent_version: 13
depends_on:
  - path: ROOT/tech_design/internal/discovery
    version: 16
  - path: ROOT/tech_design/internal/frontmatter
    version: 12
implements:
  - internal/specstaleness/specstaleness_test.go
---

# TEST/tech_design/internal/spec_staleness

## Context

Tests call `CheckSpecStaleness` with a `discovery.DiscoveredNode` and
a `map[string]*frontmatter.Frontmatter` cache.

Define these helpers in the test file:

```go
func testIntPtr(n int) *int { return &n }

func testMakeFM(
    version *int,
    parentVersion *int,
    subjectVersion *int,
    title string,
    dependsOn []frontmatter.DependsOn,
) *frontmatter.Frontmatter {
    return &frontmatter.Frontmatter{
        Version:        version,
        ParentVersion:  parentVersion,
        SubjectVersion: subjectVersion,
        Title:          title,
        DependsOn:      dependsOn,
    }
}
```

Use `testMakeFM` for every non-nil cache entry. Use `testIntPtr(n)` for
pointer fields. Pass `nil` for absent optional fields. The
`DependsOn` parameter type is `[]frontmatter.DependsOn` — pass
`nil` when there are no dependencies.

Cache keys and `FilePath` values are project-root-relative paths.
All path resolution uses real `logical_names` functions — no mocking.

## Happy Path — Spec Nodes

### All checks pass — spec node

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config", nil)`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: empty slice.

### All checks pass — root node

Cache:
- `"code-from-spec/_node.md"`:
  `testMakeFM(testIntPtr(7), nil, nil, "ROOT", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT",
FilePath: "code-from-spec/_node.md"}`.

Expect: empty slice (no parent check for root).

### All checks pass — spec node with dependencies

Cache:
- `"code-from-spec/tech_design/main/_node.md"`:
  `testMakeFM(testIntPtr(3), testIntPtr(10), nil, "ROOT/tech_design/main",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness", Version: 6}})`
- `"code-from-spec/tech_design/_node.md"`:
  `testMakeFM(testIntPtr(10), nil, nil, "ROOT/tech_design", nil)`
- `"code-from-spec/domain/staleness/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain/staleness", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/tech_design/main",
FilePath: "code-from-spec/tech_design/main/_node.md"}`.

Expect: empty slice.

## Happy Path — Test Nodes

### All checks pass — test node

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config", nil)`
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), nil, nil, "ROOT/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: empty slice.

### All checks pass — named test node

Cache:
- `"code-from-spec/domain/config/edge_cases.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config(edge_cases)", nil)`
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), nil, nil, "ROOT/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config(edge_cases)",
FilePath: "code-from-spec/domain/config/edge_cases.test.md"}`.

Expect: empty slice.

### TEST canonical vs TEST(default) treated as equal

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config(default)", nil)`
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), nil, nil, "ROOT/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: empty slice (`LogicalNamesMatch` treats these as equal).

## Blocking Steps (1-2) — Spec Nodes

### Node not in cache

Cache: empty.

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `invalid_frontmatter`.

### Node in cache with nil

Cache:
- `"code-from-spec/domain/config/_node.md"`: `nil`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `invalid_frontmatter`.

### Version missing

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(nil, testIntPtr(5), nil, "ROOT/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on non-root

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), nil, nil, "ROOT/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on root is ok

Cache:
- `"code-from-spec/_node.md"`:
  `testMakeFM(testIntPtr(7), nil, nil, "ROOT", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT",
FilePath: "code-from-spec/_node.md"}`.

Expect: empty slice.

## Blocking Steps (1-2) — Test Nodes

### Test node not in cache

Cache: empty.

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: single result with status `invalid_frontmatter`.

### Test node version missing

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(nil, nil, testIntPtr(2), "TEST/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: single result with status `invalid_frontmatter`.

### SubjectVersion missing on test node

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, nil, "TEST/domain/config", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: single result with status `invalid_frontmatter`.

## Individual Statuses — Spec Nodes

### wrong_name — title mismatch

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/old_name", nil)`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `wrong_name`.

### wrong_name — empty title

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "", nil)`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: results include `wrong_name`.

### invalid_parent — parent not in cache

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config", nil)`

`"code-from-spec/domain/_node.md"` is not in cache.

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: results include `invalid_parent`.

### invalid_parent — parent is nil in cache

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config", nil)`
- `"code-from-spec/domain/_node.md"`: `nil`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: results include `invalid_parent`.

### parent_changed

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config", nil)`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain", nil)`

ParentVersion is 5 but parent's Version is 6.

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `parent_changed`.

## Individual Statuses — Test Nodes

### invalid_subject — subject not in cache

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config", nil)`

`"code-from-spec/domain/config/_node.md"` is not in cache.

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: results include `invalid_subject`.

### invalid_subject — subject is nil in cache

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config", nil)`
- `"code-from-spec/domain/config/_node.md"`: `nil`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: results include `invalid_subject`.

### subject_changed

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/config", nil)`
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(3), nil, nil, "ROOT/domain/config", nil)`

SubjectVersion is 2 but subject's Version is 3.

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Expect: single result with status `subject_changed`.

## Dependency Statuses (Both Node Types)

### invalid_dependency — path cannot be resolved

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config",
  []frontmatter.DependsOn{{Path: "INVALID/bad", Version: 1}})`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: results include `invalid_dependency`.

### invalid_dependency — not in cache

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness", Version: 6}})`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`

`"code-from-spec/domain/staleness/_node.md"` is not in cache.

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: results include `invalid_dependency`.

### dependency_changed

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness", Version: 4}})`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`
- `"code-from-spec/domain/staleness/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain/staleness", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: single result with status `dependency_changed`.

### dependency with subsection qualifier resolved correctly

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/config",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness(interface)", Version: 6}})`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(5), nil, nil, "ROOT/domain", nil)`
- `"code-from-spec/domain/staleness/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain/staleness", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: empty slice (qualifier stripped, file found at correct version).

## Accumulation

### Multiple problems collected — spec node

Cache:
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(2), testIntPtr(5), nil, "ROOT/domain/old_name",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness", Version: 4}})`
- `"code-from-spec/domain/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain", nil)`
- `"code-from-spec/domain/staleness/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain/staleness", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Three problems: wrong title, parent changed (5 vs 6), dependency
changed (4 vs 6).

Expect: three results with statuses `wrong_name`, `parent_changed`,
`dependency_changed`.

### Multiple problems collected — test node

Cache:
- `"code-from-spec/domain/config/default.test.md"`:
  `testMakeFM(testIntPtr(1), nil, testIntPtr(2), "TEST/domain/wrong",
  []frontmatter.DependsOn{{Path: "ROOT/domain/staleness", Version: 4}})`
- `"code-from-spec/domain/config/_node.md"`:
  `testMakeFM(testIntPtr(3), nil, nil, "ROOT/domain/config", nil)`
- `"code-from-spec/domain/staleness/_node.md"`:
  `testMakeFM(testIntPtr(6), nil, nil, "ROOT/domain/staleness", nil)`

Node: `discovery.DiscoveredNode{LogicalName: "TEST/domain/config",
FilePath: "code-from-spec/domain/config/default.test.md"}`.

Three problems: wrong title, subject changed (2 vs 3), dependency
changed (4 vs 6).

Expect: three results with statuses `wrong_name`, `subject_changed`,
`dependency_changed`.

### Blocking step prevents accumulation

Cache:
- `"code-from-spec/domain/config/_node.md"`: `nil`

Node: `discovery.DiscoveredNode{LogicalName: "ROOT/domain/config",
FilePath: "code-from-spec/domain/config/_node.md"}`.

Expect: exactly one result with status `invalid_frontmatter`.
