---
version: 5
parent_version: 11
depends_on:
  - path: ROOT/domain/staleness
    version: 6
  - path: ROOT/domain/name_verification
    version: 2
  - path: ROOT/domain/output
    version: 6
  - path: ROOT/tech_design/logical_names
    version: 4
implements:
  - cmd/staleness-check/specstaleness.go
---

# ROOT/tech_design/spec_staleness

## Intent

Verifies spec staleness for a single node. The caller
invokes this function once per discovered node (spec or
test) and collects the results.

## Contracts

### Interface

```go
type StalenessResult struct {
    Node   string
    File   string
    Status string
}

func CheckSpecStaleness(
    node DiscoveredNode,
    cache map[string]*Frontmatter,
) []StalenessResult
```

`CheckSpecStaleness` checks one node for spec staleness.
Returns an empty slice if the node is not stale. Returns
one `StalenessResult` per problem found — a node may have
multiple problems (e.g., wrong name, parent changed, and
dependency changed simultaneously).

The `cache` maps file paths to parsed frontmatters,
populated by the caller before invoking this function.
Every discovered node has an entry in the cache: a valid
`*Frontmatter` on success, or `nil` if frontmatter parsing failed.
If a file path has no entry in the cache, the file does
not exist.

### Algorithm

Check in this order. Steps 1-2 are blocking — if they
fail, return immediately with a single result. From step
3 onward, collect all problems found.

1. Look up the node's frontmatter in the cache. If not
   found or nil → return `[invalid_frontmatter]`.
2. Check required fields: `version` must be present;
   `parent_version` must be present for non-root nodes.
   If missing → return `[invalid_frontmatter]`.
3. Use `LogicalNamesMatch` to compare the frontmatter
   `Title` against the node's `LogicalName`. If it does
   not match or Title is empty → collect `wrong_name`.
4. Parent check: use `HasParent` to determine if the
   node has a parent. If it does:
   - Use `ParentLogicalName` to get the parent's logical
     name, then `PathFromLogicalName` to get its file
     path.
   - Look up the parent's frontmatter in the cache. If
     not found or nil → collect `invalid_parent`.
   - Otherwise compare: if
     `node.parent_version != parent.version` → collect
     `parent_changed`.
5. Dependency check (for each `depends_on` entry):
   - Use `PathFromLogicalName` to resolve the
     dependency's file path from its logical name.
   - Look up the dependency's frontmatter in the cache.
     If the path cannot be resolved, not found, or nil
     → collect `invalid_dependency`.
   - Otherwise compare: if `depends_on.version !=
     dependency.version` → collect `dependency_changed`.
6. Return all collected results (empty slice if none).

### Path resolution

Parent and dependency logical names are resolved to file
paths using `HasParent`, `ParentLogicalName`, and
`PathFromLogicalName` from `ROOT/tech_design/logical_names`.

