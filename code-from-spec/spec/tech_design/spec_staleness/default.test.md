---
version: 2
parent_version: 5
implements:
  - cmd/staleness-check/specstaleness_test.go
---

# TEST/tech_design/spec_staleness

## Context

Tests build a `cache` (`map[string]*Frontmatter`) with
controlled entries and call `CheckSpecStaleness` with a
`DiscoveredNode`. `Version` and `ParentVersion` in
`Frontmatter` are `*int` — use helper to create pointer
values. All path resolution uses real `logical_names`
functions — no mocking.

## Happy Path

### All checks pass — spec node

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: empty slice.

### All checks pass — test node

Cache contains:
- `spec/domain/config/default.test.md`: Version=1,
  ParentVersion=2, Title=`"TEST/domain/config"`
- `spec/domain/config/_node.md`: Version=2,
  Title=`"ROOT/domain/config"`

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"spec/domain/config/default.test.md"`.

Expect: empty slice.

### All checks pass — root node

Cache contains:
- `spec/_node.md`: Version=7, Title=`"ROOT"`

Node: LogicalName=`"ROOT"`,
FilePath=`"spec/_node.md"`.

Expect: empty slice (no parent check for root).

### All checks pass — with dependencies

Cache contains:
- `spec/tech_design/main/_node.md`: Version=3,
  ParentVersion=10, Title=`"ROOT/tech_design/main"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:6},
  {Path:"EXTERNAL/api", Version:2}]
- `spec/tech_design/_node.md`: Version=10,
  Title=`"ROOT/tech_design"`
- `spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`
- `external/api/_external.md`: Version=2,
  Title=`"EXTERNAL/api"`

Node: LogicalName=`"ROOT/tech_design/main"`,
FilePath=`"spec/tech_design/main/_node.md"`.

Expect: empty slice.

## Blocking Steps (1-2)

### Node not in cache

Cache is empty.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### Node in cache with nil

Cache contains:
- `spec/domain/config/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### Version missing

Cache contains:
- `spec/domain/config/_node.md`: Version=nil,
  ParentVersion=5, Title=`"ROOT/domain/config"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on non-root

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=nil, Title=`"ROOT/domain/config"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on root is ok

Cache contains:
- `spec/_node.md`: Version=7, ParentVersion=nil,
  Title=`"ROOT"`

Node: LogicalName=`"ROOT"`,
FilePath=`"spec/_node.md"`.

Expect: empty slice (root does not need parent_version).

## Individual Statuses

### wrong_name — title mismatch

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/old_name"`
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with status `wrong_name`.

### wrong_name — empty title

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`""`
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: results include `wrong_name`.

### wrong_name — TEST canonical vs TEST(default)

Cache contains:
- `spec/domain/config/default.test.md`: Version=1,
  ParentVersion=2,
  Title=`"TEST/domain/config(default)"`
- `spec/domain/config/_node.md`: Version=2,
  Title=`"ROOT/domain/config"`

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"spec/domain/config/default.test.md"`.

Expect: empty slice (LogicalNamesMatch treats these as
equal).

### invalid_parent — parent not in cache

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Parent `spec/domain/_node.md` is not in cache.

Expect: results include `invalid_parent`.

### invalid_parent — parent is nil in cache

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `spec/domain/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: results include `invalid_parent`.

### parent_changed

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `spec/domain/_node.md`: Version=6,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

ParentVersion is 5 but parent's Version is 6.

Expect: single result with status `parent_changed`.

### invalid_dependency — path cannot be resolved

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"INVALID/bad", Version:1}]
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: results include `invalid_dependency`.

### invalid_dependency — not in cache

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:6}]
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

`spec/domain/staleness/_node.md` is not in cache.

Expect: results include `invalid_dependency`.

### invalid_dependency — nil in cache

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:6}]
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`
- `spec/domain/staleness/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: results include `invalid_dependency`.

### dependency_changed

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4}]
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`
- `spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

DependsOn version is 4 but dependency's Version is 6.

Expect: single result with status `dependency_changed`.

## Accumulation

### Multiple problems collected

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/old_name"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4}]
- `spec/domain/_node.md`: Version=6,
  Title=`"ROOT/domain"`
- `spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Three problems: wrong title, parent changed (5 vs 6),
dependency changed (4 vs 6).

Expect: three results with statuses `wrong_name`,
`parent_changed`, `dependency_changed`.

### Multiple dependency problems

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4},
  {Path:"ROOT/domain/output", Version:3}]
- `spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`
- `spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

First dependency changed (4 vs 6), second dependency
not in cache.

Expect: two results with statuses `dependency_changed`
and `invalid_dependency`.

### Blocking step prevents accumulation

Cache contains:
- `spec/domain/config/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: exactly one result with status
`invalid_frontmatter` — no further checks performed.
