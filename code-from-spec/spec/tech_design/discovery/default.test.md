---
version: 5
parent_version: 9
implements:
  - cmd/staleness-check/discovery_test.go
---

# TEST/tech_design/discovery

## Context

Each test uses `t.TempDir()` to create an isolated
temporary directory representing the project root. The
test creates the required file structure inside it, then
calls `os.Chdir()` to set it as the working directory
before invoking `DiscoverNodes`. Save and restore the
original working directory to avoid interference between
tests.

## Happy Path

### Discovers spec nodes at all levels

Create:
```
code-from-spec/_node.md
code-from-spec/domain/_node.md
code-from-spec/domain/config/_node.md
```

Expect `specNodes`:
- `ROOT` → `code-from-spec/_node.md`
- `ROOT/domain` → `code-from-spec/domain/_node.md`
- `ROOT/domain/config` → `code-from-spec/domain/config/_node.md`

### Discovers test nodes

Create:
```
code-from-spec/domain/config/_node.md
code-from-spec/domain/config/default.test.md
code-from-spec/domain/config/edge_cases.test.md
```

Expect `testNodes`:
- `TEST/domain/config` → `code-from-spec/domain/config/default.test.md`
- `TEST/domain/config(edge_cases)` → `code-from-spec/domain/config/edge_cases.test.md`

### Test nodes alongside intermediate nodes

Create:
```
code-from-spec/domain/_node.md
code-from-spec/domain/default.test.md
code-from-spec/domain/config/_node.md
```

Expect `testNodes` includes:
- `TEST/domain` → `code-from-spec/domain/default.test.md`

### Results are sorted alphabetically

Create a tree where natural filesystem order differs from
alphabetical order. Verify both lists are sorted by
`LogicalName`.

## Edge Cases

### Empty code-from-spec directory

Create `code-from-spec/` with no `_node.md` files.
Expect an error returned.

### Non-node files are ignored

Create files under `code-from-spec/` that are not
`_node.md` or `*.test.md` (e.g., `README.md`,
`notes.txt`). Expect them to be absent from all lists.

## Failure Cases

### code-from-spec directory does not exist

Do not create `code-from-spec/`. Expect an error
returned.
