---
version: 12
parent_version: 2
depends_on:
  - path: ROOT/domain/output
    version: 7
  - path: ROOT/domain/staleness
    version: 7
  - path: ROOT/tech_design/internal/logical_names
    version: 9
  - path: ROOT/tech_design/internal/spec_comment
    version: 12
  - path: ROOT/tech_design/internal/spec_staleness
    version: 11
implements:
  - internal/codestaleness/codestaleness.go
---

# ROOT/tech_design/internal/code_staleness

## Intent

Verifies code staleness for a single node. The caller
invokes this function once per discovered node and
collects the results.

## Contracts

### Package

`codestaleness`

### Interface

```go
func CheckCodeStaleness(
    node DiscoveredNode,
    cache map[string]*Frontmatter,
) []specstaleness.StalenessResult
```

`CheckCodeStaleness` checks one node for code staleness.
Returns an empty slice if all files are up to date or
the node has no `implements`. Returns one
`specstaleness.StalenessResult` per problem found.

The `cache` maps file paths to parsed frontmatters,
populated by the caller before invoking this function.
Every discovered node has an entry in the cache: a valid
`*Frontmatter` on success, or `nil` if frontmatter
parsing failed. If a file path has no entry in the
cache, the file does not exist.

### Algorithm

Check in this order. Steps 1-3 are blocking — return
immediately with a single result. Step 4 produces one
result per problematic file.

1. Look up the node's frontmatter in the cache. If not
   found or nil → return `[unreadable_frontmatter]`.
2. Check that `Version` is not nil. If nil → return
   `[no_version]`.
3. If `Implements` is empty → return empty slice.
4. For each file in `Implements`, produce at most one
   `specstaleness.StalenessResult` with `Node` = the node's logical
   name, `File` = the file path, and `Status` set by
   the first matching condition:
   - File does not exist → `missing`.
   - `speccomment.ParseSpecComment` returns an error where
     `errors.Is(err, speccomment.ErrNoSpecComment)` →
     `no_spec_comment`.
   - `speccomment.ParseSpecComment` returns an error where
     `errors.Is(err, speccomment.ErrMalformed)` →
     `malformed_spec_comment`.
   - `LogicalNamesMatch` between the spec comment's
     logical name and the node's `LogicalName` returns
     false → `wrong_node`.
   - `*node.Version != spec_comment.Version` → `stale`.
   - None of the above → file is up to date, omit from
     results.
