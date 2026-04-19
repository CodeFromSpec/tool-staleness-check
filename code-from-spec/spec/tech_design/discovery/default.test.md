---
version: 2
parent_version: 5
implements:
  - cmd/staleness-check/discovery_test.go
---

# TEST/tech_design/discovery

## Context

Each test uses `t.TempDir()` to create an isolated
temporary directory. The test creates the required file
structure inside it, then calls `os.Chdir()` to set it
as the working directory before invoking `DiscoverNodes`.
Save and restore the original working directory to avoid
interference between tests.

## Happy Path

### Discovers spec nodes at all levels

Create:
```
spec/_node.md
spec/domain/_node.md
spec/domain/config/_node.md
```

Expect `specNodes`:
- `ROOT` → `spec/_node.md`
- `ROOT/domain` → `spec/domain/_node.md`
- `ROOT/domain/config` → `spec/domain/config/_node.md`

### Discovers test nodes

Create:
```
spec/domain/config/_node.md
spec/domain/config/default.test.md
spec/domain/config/edge_cases.test.md
```

Expect `testNodes`:
- `TEST/domain/config` → `spec/domain/config/default.test.md`
- `TEST/domain/config(edge_cases)` → `spec/domain/config/edge_cases.test.md`

### Discovers external dependencies

Create:
```
external/database/_external.md
external/celcoin-api/_external.md
```

Expect `externalDeps`:
- `EXTERNAL/celcoin-api` → `external/celcoin-api/_external.md`
- `EXTERNAL/database` → `external/database/_external.md`

### Results are sorted alphabetically

Create a tree where natural filesystem order differs from
alphabetical order. Verify all three lists are sorted by
`LogicalName`.

## Edge Cases

### Empty spec directory

Create `spec/` with no files. Expect empty `specNodes`,
empty `testNodes`. No error.

### Empty external directory

Create `external/` with no subdirectories. Expect empty
`externalDeps`. No error.

### Non-node files are ignored

Create files in `spec/` that are not `_node.md` or
`*.test.md` (e.g., `README.md`, `notes.txt`). Expect
them to be absent from all lists.

### External directory with extra files

Create `external/database/_external.md` alongside other
files (`schema.sql`, `README.md`). Expect only one
`externalDeps` entry for `EXTERNAL/database`.

## Failure Cases

### spec directory does not exist

Do not create `spec/`. Expect an error returned.

### external directory does not exist

Do not create `external/`. Spec and test discovery
should still succeed. `externalDeps` is empty. No error.
