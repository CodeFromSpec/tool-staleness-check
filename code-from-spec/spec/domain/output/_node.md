---
version: 7
parent_version: 10
---

# ROOT/domain/output

## Intent

Defines the output structure and status values produced
by the staleness verification tool.

## Contracts

### Output sections

The tool produces three sections in order:

1. **Spec staleness** — stale spec nodes (`_node.md`),
   sorted alphabetically by logical name.
2. **Test staleness** — stale test nodes (`*.test.md`),
   sorted alphabetically by logical name.
3. **Code staleness** — stale generated files, sorted
   by node logical name, then by file path.

Only nodes or files with problems are included. Nodes
and files that pass all checks are omitted. If a section
has no problems, it is an empty list `[]`.

### Spec and test staleness statuses

| Status | Applies to | Condition |
|---|---|---|
| `invalid_frontmatter` | Both | Frontmatter cannot be parsed or is missing required fields. |
| `wrong_name` | Both | Title does not match expected logical name. |
| `invalid_parent` | Spec nodes | Parent file cannot be found or read. |
| `parent_changed` | Spec nodes | `node.parent_version != parent.version`. |
| `invalid_subject` | Test nodes | Subject file cannot be found or read. |
| `subject_changed` | Test nodes | `node.subject_version != subject.version`. |
| `invalid_dependency` | Both | `depends_on` entry is malformed or referenced file cannot be found or read. |
| `dependency_changed` | Both | `depends_on[].version != dependency.version`. |

### Code staleness statuses

| Status | Condition |
|---|---|
| `unreadable_frontmatter` | Frontmatter cannot be parsed. |
| `no_version` | Frontmatter has no `version` field. |
| `missing` | File in `implements` does not exist. |
| `no_spec_comment` | File exists but has no spec comment. |
| `malformed_spec_comment` | Spec comment exists but cannot be parsed. |
| `wrong_node` | Spec comment references a different node. |
| `stale` | `node.version != spec comment version`. |

### Entry format

Spec and test staleness entries have `node` and `statuses`.
The `statuses` field is a list — a node may have multiple
problems detected in a single run. If the frontmatter
cannot be parsed, only `invalid_frontmatter` is reported
(further checks require a valid frontmatter). Otherwise,
all applicable statuses are collected in the list.

Code staleness entries have `node`, `file`, and `status`.
The `status` field is a single string — each file has at
most one problem (checks are sequential prerequisites).

### Examples

All nodes up to date — all sections empty:

```yaml
spec_staleness: []
test_staleness: []
code_staleness: []
```

Mixed staleness — spec node with multiple problems,
test node with subject changed, generated file stale:

```yaml
spec_staleness:
  - node: ROOT/domain/api
    statuses:
      - wrong_name
      - parent_changed
      - dependency_changed
  - node: ROOT/domain/config
    statuses:
      - parent_changed
test_staleness:
  - node: TEST/domain/config
    statuses:
      - subject_changed
code_staleness:
  - node: ROOT/domain/config
    file: internal/config/config.go
    status: stale
  - node: TEST/domain/config
    file: internal/config/config_test.go
    status: stale
```

Error conditions — missing files, bad frontmatter,
wrong name:

```yaml
spec_staleness:
  - node: ROOT/domain
    statuses:
      - invalid_frontmatter
  - node: ROOT/domain/config
    statuses:
      - invalid_parent
      - dependency_changed
test_staleness: []
code_staleness:
  - node: ROOT/domain/api
    file: internal/api/router.go
    status: missing
  - node: ROOT/domain/api
    file: internal/api/handler.go
    status: no_spec_comment
```

Named test node stale:

```yaml
spec_staleness: []
test_staleness:
  - node: TEST/domain/config(edge_cases)
    statuses:
      - subject_changed
code_staleness:
  - node: TEST/domain/config(edge_cases)
    file: internal/config/config_edge_test.go
    status: stale
```

### Exit codes

| Code | Meaning |
|---|---|
| 0 | All sections are empty (no problems found). |
| 1 | At least one entry exists (problems found). |
| 2 | Operational error (e.g., cannot read filesystem). |
