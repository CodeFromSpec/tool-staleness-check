---
version: 8
parent_version: 11
implements:
  - internal/specstaleness/specstaleness_test.go
---

# TEST/tech_design/internal/spec_staleness

## Context

Tests build a `cache` (`map[string]*Frontmatter`) with
controlled entries and call `CheckSpecStaleness` with a
`DiscoveredNode`. `Version`, `ParentVersion`, and
`SubjectVersion` in `Frontmatter` are `*int` — use helper
to create pointer values. All path resolution uses real
`logical_names` functions — no mocking.

Cache keys and `DiscoveredNode.FilePath` values are
project-root-relative paths (e.g.,
`code-from-spec/domain/config/_node.md`).

## Happy Path — Spec Nodes

### All checks pass — spec node

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"code-from-spec/domain/config/_node.md"`.

Expect: empty slice.

### All checks pass — root node

Cache contains:
- `code-from-spec/_node.md`: Version=7, Title=`"ROOT"`

Node: LogicalName=`"ROOT"`,
FilePath=`"code-from-spec/_node.md"`.

Expect: empty slice (no parent check for root).

### All checks pass — spec node with dependencies

Cache contains:
- `code-from-spec/tech_design/main/_node.md`: Version=3,
  ParentVersion=10, Title=`"ROOT/tech_design/main"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:6}]
- `code-from-spec/tech_design/_node.md`: Version=10,
  Title=`"ROOT/tech_design"`
- `code-from-spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Node: LogicalName=`"ROOT/tech_design/main"`,
FilePath=`"code-from-spec/tech_design/main/_node.md"`.

Expect: empty slice.

## Happy Path — Test Nodes

### All checks pass — test node

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config"`
- `code-from-spec/domain/config/_node.md`: Version=2,
  Title=`"ROOT/domain/config"`

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"code-from-spec/domain/config/default.test.md"`.

Expect: empty slice.

### All checks pass — named test node

Cache contains:
- `code-from-spec/domain/config/edge_cases.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config(edge_cases)"`
- `code-from-spec/domain/config/_node.md`: Version=2,
  Title=`"ROOT/domain/config"`

Node: LogicalName=`"TEST/domain/config(edge_cases)"`,
FilePath=`"code-from-spec/domain/config/edge_cases.test.md"`.

Expect: empty slice.

### wrong_name — TEST canonical vs TEST(default)

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config(default)"`
- `code-from-spec/domain/config/_node.md`: Version=2,
  Title=`"ROOT/domain/config"`

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"code-from-spec/domain/config/default.test.md"`.

Expect: empty slice (LogicalNamesMatch treats these as
equal).

## Blocking Steps (1-2) — Spec Nodes

### Node not in cache

Cache is empty.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"code-from-spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### Node in cache with nil

Cache contains:
- `code-from-spec/domain/config/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"code-from-spec/domain/config/_node.md"`.

Expect: single result with status `invalid_frontmatter`.

### Version missing

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=nil,
  ParentVersion=5, Title=`"ROOT/domain/config"`

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on non-root

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=nil, Title=`"ROOT/domain/config"`

Expect: single result with status `invalid_frontmatter`.

### ParentVersion missing on root is ok

Cache contains:
- `code-from-spec/_node.md`: Version=7,
  ParentVersion=nil, Title=`"ROOT"`

Node: LogicalName=`"ROOT"`,
FilePath=`"code-from-spec/_node.md"`.

Expect: empty slice.

## Blocking Steps (1-2) — Test Nodes

### Test node not in cache

Cache is empty.

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"code-from-spec/domain/config/default.test.md"`.

Expect: single result with status `invalid_frontmatter`.

### Test node version missing

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=nil, SubjectVersion=2,
  Title=`"TEST/domain/config"`

Expect: single result with status `invalid_frontmatter`.

### SubjectVersion missing on test node

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=nil,
  Title=`"TEST/domain/config"`

Expect: single result with status `invalid_frontmatter`.

## Individual Statuses — Spec Nodes

### wrong_name — title mismatch

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/old_name"`
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"code-from-spec/domain/config/_node.md"`.

Expect: single result with status `wrong_name`.

### wrong_name — empty title

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`""`
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Expect: results include `wrong_name`.

### invalid_parent — parent not in cache

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`

`code-from-spec/domain/_node.md` is not in cache.

Expect: results include `invalid_parent`.

### invalid_parent — parent is nil in cache

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `code-from-spec/domain/_node.md`: nil

Expect: results include `invalid_parent`.

### parent_changed

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`
- `code-from-spec/domain/_node.md`: Version=6,
  Title=`"ROOT/domain"`

ParentVersion is 5 but parent's Version is 6.

Expect: single result with status `parent_changed`.

## Individual Statuses — Test Nodes

### invalid_subject — subject not in cache

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config"`

`code-from-spec/domain/config/_node.md` is not in cache.

Expect: results include `invalid_subject`.

### invalid_subject — subject is nil in cache

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config"`
- `code-from-spec/domain/config/_node.md`: nil

Expect: results include `invalid_subject`.

### subject_changed

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/config"`
- `code-from-spec/domain/config/_node.md`: Version=3,
  Title=`"ROOT/domain/config"`

SubjectVersion is 2 but subject's Version is 3.

Expect: single result with status `subject_changed`.

## Dependency Statuses (Both Node Types)

### invalid_dependency — path cannot be resolved

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"INVALID/bad", Version:1}]
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

Expect: results include `invalid_dependency`.

### invalid_dependency — not in cache

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:6}]
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`

`code-from-spec/domain/staleness/_node.md` is not in
cache.

Expect: results include `invalid_dependency`.

### dependency_changed

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4}]
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`
- `code-from-spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Expect: single result with status `dependency_changed`.

### dependency with subsection qualifier resolved correctly

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/config"`,
  DependsOn=[{Path:"ROOT/domain/staleness(interface)",
  Version:6}]
- `code-from-spec/domain/_node.md`: Version=5,
  Title=`"ROOT/domain"`
- `code-from-spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Expect: empty slice (qualifier stripped, file found at
correct version).

## Accumulation

### Multiple problems collected — spec node

Cache contains:
- `code-from-spec/domain/config/_node.md`: Version=2,
  ParentVersion=5, Title=`"ROOT/domain/old_name"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4}]
- `code-from-spec/domain/_node.md`: Version=6,
  Title=`"ROOT/domain"`
- `code-from-spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Three problems: wrong title, parent changed (5 vs 6),
dependency changed (4 vs 6).

Expect: three results with statuses `wrong_name`,
`parent_changed`, `dependency_changed`.

### Multiple problems collected — test node

Cache contains:
- `code-from-spec/domain/config/default.test.md`:
  Version=1, SubjectVersion=2,
  Title=`"TEST/domain/wrong"`,
  DependsOn=[{Path:"ROOT/domain/staleness", Version:4}]
- `code-from-spec/domain/config/_node.md`: Version=3,
  Title=`"ROOT/domain/config"`
- `code-from-spec/domain/staleness/_node.md`: Version=6,
  Title=`"ROOT/domain/staleness"`

Three problems: wrong title, subject changed (2 vs 3),
dependency changed (4 vs 6).

Expect: three results with statuses `wrong_name`,
`subject_changed`, `dependency_changed`.

### Blocking step prevents accumulation

Cache contains:
- `code-from-spec/domain/config/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"code-from-spec/domain/config/_node.md"`.

Expect: exactly one result with status
`invalid_frontmatter`.
