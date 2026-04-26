---
version: 14
parent_version: 2
depends_on:
  - path: ROOT/domain/specifications
    version: 4
  - path: ROOT/tech_design/internal/logical_names
    version: 9
implements:
  - internal/discovery/discovery.go
---

# ROOT/tech_design/internal/discovery

## Intent

Walks the filesystem to discover all spec nodes and test
nodes.

## Contracts

### Package

`discovery`

### Discovery rules

Walk `code-from-spec/` recursively:
- Every `_node.md` file produces a spec node.
- Every `*.test.md` file produces a test node.

For each discovered file, use `LogicalNameFromPath` from
`ROOT/tech_design/logical_names` to derive the logical
name. Paths passed to `LogicalNameFromPath` are relative
to the project root.

### Interface

```go
type DiscoveredNode struct {
    LogicalName string
    FilePath    string
}

func DiscoverNodes() (
    specNodes []DiscoveredNode,
    testNodes []DiscoveredNode,
    err       error,
)
```

All lists are sorted alphabetically by `LogicalName`.
`FilePath` values are relative to the project root
(e.g., `code-from-spec/domain/config/_node.md`).

### Error handling

Errors returned by `DiscoverNodes` must wrap the
underlying error with a descriptive message so the
caller can print it directly. Examples:

- `code-from-spec/ directory not found: <underlying error>`
- `error walking code-from-spec/ directory: <underlying error>`

If `code-from-spec/` does not exist or contains no
`_node.md` files, return an error.
