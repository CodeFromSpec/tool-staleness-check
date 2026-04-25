---
version: 2
parent_version: 9
implements:
  - cmd/staleness-check/main_test.go
---

# TEST/tech_design/main

## Context

Tests exercise the full pipeline by creating a temporary
directory structure with controlled spec nodes, external
dependencies, and generated files. The tool is tested as
a compiled binary invoked via `os/exec`. Build the binary
once in `TestMain` and reuse it across tests.

Each test creates its own `t.TempDir()` with a
`code-from-spec/` subdirectory containing the necessary
`spec/` and optionally `external/` trees. The binary is
invoked from the `code-from-spec/` directory.

Helper functions create node files with controlled
frontmatter and generated files with controlled spec
comments.

## Help Message

### Any argument prints help

Invoke the binary with `--help`. Expect exit code 0 and
stdout containing `staleness-check` and `Usage`.

### Different argument prints help

Invoke the binary with `foo`. Expect exit code 0 and
stdout containing `staleness-check` and `Usage`.

## Happy Path

### All nodes up to date

Create a minimal spec tree:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=1, parent_version=1,
  title=`ROOT/domain`

No implements, no dependencies.

Expect exit code 0 and YAML output:
```yaml
spec_staleness: []
test_staleness: []
code_staleness: []
```

### Node with up-to-date generated file

Create:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=2, parent_version=1,
  title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `cmd/staleness-check/gen.go` with
  `// spec: ROOT/domain@v2`

Expect exit code 0, all sections empty.

### Node with dependencies all current

Create:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=3, parent_version=1,
  title=`ROOT/domain`
- `spec/domain/config/_node.md`: version=1,
  parent_version=3, title=`ROOT/domain/config`,
  depends_on=[{path: ROOT/domain, version: 3}]

Expect exit code 0, all sections empty.

## Spec Staleness

### Parent changed

Create:
- `spec/_node.md`: version=2, title=`ROOT`
- `spec/domain/_node.md`: version=1, parent_version=1,
  title=`ROOT/domain`

parent_version=1 but parent version=2.

Expect exit code 1. spec_staleness contains one entry
with node=`ROOT/domain` and statuses including
`parent_changed`.

### Multiple statuses on one node

Create:
- `spec/_node.md`: version=2, title=`ROOT`
- `spec/domain/_node.md`: version=1, parent_version=1,
  title=`ROOT/domain/wrong`,
  depends_on=[{path: ROOT/missing, version: 1}]

Wrong title, parent changed, invalid dependency.

Expect exit code 1. spec_staleness contains one entry
with node=`ROOT/domain` and statuses including
`wrong_name`, `parent_changed`, `invalid_dependency`.

## Test Staleness

### Test node parent changed

Create:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=2, parent_version=1,
  title=`ROOT/domain`
- `spec/domain/default.test.md`: version=1,
  parent_version=1, title=`TEST/domain`

Test's parent_version=1 but parent (domain) version=2.

Expect exit code 1. test_staleness contains one entry
with node=`TEST/domain` and statuses including
`parent_changed`.

## Code Staleness

### Generated file is stale

Create:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=3, parent_version=1,
  title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `cmd/staleness-check/gen.go` with
  `// spec: ROOT/domain@v2`

Version is 3 but spec comment says v2.

Expect exit code 1. code_staleness contains one entry
with node=`ROOT/domain`, file containing `gen.go`,
status=`stale`.

### Generated file missing

Create:
- `spec/_node.md`: version=1, title=`ROOT`
- `spec/domain/_node.md`: version=1, parent_version=1,
  title=`ROOT/domain`,
  implements=[cmd/staleness-check/nonexistent.go]

Do not create the file.

Expect exit code 1. code_staleness contains one entry
with status=`missing`.

## Mixed Results

### Spec, test, and code staleness together

Create:
- `spec/_node.md`: version=2, title=`ROOT`
- `spec/domain/_node.md`: version=3, parent_version=1,
  title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `spec/domain/default.test.md`: version=1,
  parent_version=1, title=`TEST/domain`,
  implements=[cmd/staleness-check/gen_test.go]
- `cmd/staleness-check/gen.go` with
  `// spec: ROOT/domain@v2`
- `cmd/staleness-check/gen_test.go` with
  `// spec: TEST/domain@v1`

Spec staleness: ROOT/domain has parent_changed (1 vs 2).
Test staleness: TEST/domain has parent_changed (1 vs 3).
Code staleness: gen.go is stale (v2 vs v3).
gen_test.go is up to date (v1 matches).

Expect exit code 1. All three sections have entries.

## Operational Error

### spec/ directory missing

Create a `code-from-spec/` directory with no `spec/`
subdirectory.

Expect exit code 2 and stderr containing an error
message.

## Output Ordering

### Nodes sorted alphabetically

Create:
- `spec/_node.md`: version=2, title=`ROOT`
- `spec/domain/_node.md`: version=1, parent_version=1,
  title=`ROOT/domain`
- `spec/arch/_node.md`: version=1, parent_version=1,
  title=`ROOT/arch`

Both have parent_changed (1 vs 2).

Expect spec_staleness entries in order: `ROOT/arch`
before `ROOT/domain`.
