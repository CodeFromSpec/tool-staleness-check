---
version: 5
parent_version: 10
depends_on:
  - path: ROOT/domain/specifications
    version: 2
  - path: ROOT/domain/external_dependencies
    version: 4
  - path: ROOT/tech_design/logical_names
    version: 3
implements:
  - cmd/staleness-check/discovery.go
---

# ROOT/tech_design/discovery

## Intent

Walks the filesystem to discover all spec nodes, test
nodes, and external dependencies.

## Contracts

### Discovery rules

Walk `code-from-spec/spec/` recursively:
- Every `_node.md` file produces a spec node.
- Every `*.test.md` file produces a test node.

Walk `code-from-spec/external/` one level deep:
- Every `_external.md` file produces an external
  dependency.

For each discovered file, use `LogicalNameFromPath` from
`ROOT/tech_design/logical_names` to derive the logical
name.

### Interface

```go
type DiscoveredNode struct {
    LogicalName string
    FilePath    string
}

func DiscoverNodes() (
    specNodes    []DiscoveredNode,
    testNodes    []DiscoveredNode,
    externalDeps []DiscoveredNode,
    err          error,
)
```

All three lists are sorted alphabetically by
`LogicalName`.

### Error handling

Errors returned by `DiscoverNodes` must wrap the
underlying error with a descriptive message so the
caller can print it directly. Examples:

- `code-from-spec/spec/ directory not found: <underlying error>`
- `error walking code-from-spec/spec/ directory: <underlying error>`

Directories under `code-from-spec/external/` that do not contain an
`_external.md` file are silently skipped — they are not
an error.
