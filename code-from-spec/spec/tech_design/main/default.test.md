---
version: 4
parent_version: 11
implements:
  - cmd/staleness-check/main_test.go
---

# TEST/tech_design/main

## Context

Tests exercise the full pipeline by creating a temporary
directory structure with controlled spec nodes and
generated files. The tool is tested as a compiled binary
invoked via `os/exec`. Build the binary once in
`TestMain` and reuse it across tests.

Each test creates its own `t.TempDir()` representing the
project root. Spec nodes are created under
`code-from-spec/` within it. The binary is invoked from
the TempDir (the project root).

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
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=1,
  parent_version=1, title=`ROOT/domain`

No implements, no dependencies.

Expect exit code 0 and YAML output:
```yaml
spec_staleness: []
test_staleness: []
code_staleness: []
```

### Node with up-to-date generated file

Create:
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=2,
  parent_version=1, title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `cmd/staleness-check/gen.go` with
  `// code-from-spec: ROOT/domain@v2`

Expect exit code 0, all sections empty.

### Node with dependencies all current

Create:
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=3,
  parent_version=1, title=`ROOT/domain`
- `code-from-spec/domain/config/_node.md`: version=1,
  parent_version=3, title=`ROOT/domain/config`,
  depends_on=[{path: ROOT/domain, version: 3}]

Expect exit code 0, all sections empty.

## Spec Staleness

### Parent changed

Create:
- `code-from-spec/_node.md`: version=2, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=1,
  parent_version=1, title=`ROOT/domain`

parent_version=1 but parent version=2.

Expect exit code 1. spec_staleness contains one entry
with node=`ROOT/domain` and statuses including
`parent_changed`.

### Multiple statuses on one node

Create:
- `code-from-spec/_node.md`: version=2, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=1,
  parent_version=1, title=`ROOT/domain/wrong`,
  depends_on=[{path: ROOT/missing, version: 1}]

Wrong title, parent changed, invalid dependency.

Expect exit code 1. spec_staleness contains one entry
with node=`ROOT/domain` and statuses including
`wrong_name`, `parent_changed`, `invalid_dependency`.

## Test Staleness

### Test node subject changed

Create:
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=2,
  parent_version=1, title=`ROOT/domain`
- `code-from-spec/domain/default.test.md`: version=1,
  subject_version=1, title=`TEST/domain`

Test's subject_version=1 but subject (domain) version=2.

Expect exit code 1. test_staleness contains one entry
with node=`TEST/domain` and statuses including
`subject_changed`.

## Code Staleness

### Generated file is stale

Create:
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=3,
  parent_version=1, title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `cmd/staleness-check/gen.go` with
  `// code-from-spec: ROOT/domain@v2`

Version is 3 but spec comment says v2.

Expect exit code 1. code_staleness contains one entry
with node=`ROOT/domain`, file containing `gen.go`,
status=`stale`.

### Generated file missing

Create:
- `code-from-spec/_node.md`: version=1, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=1,
  parent_version=1, title=`ROOT/domain`,
  implements=[cmd/staleness-check/nonexistent.go]

Do not create the file.

Expect exit code 1. code_staleness contains one entry
with status=`missing`.

## Mixed Results

### Spec, test, and code staleness together

Create:
- `code-from-spec/_node.md`: version=2, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=3,
  parent_version=1, title=`ROOT/domain`,
  implements=[cmd/staleness-check/gen.go]
- `code-from-spec/domain/default.test.md`: version=1,
  subject_version=1, title=`TEST/domain`,
  implements=[cmd/staleness-check/gen_test.go]
- `cmd/staleness-check/gen.go` with
  `// code-from-spec: ROOT/domain@v2`
- `cmd/staleness-check/gen_test.go` with
  `// code-from-spec: TEST/domain@v1`

Spec staleness: ROOT/domain has parent_changed (1 vs 2).
Test staleness: TEST/domain has subject_changed (1 vs 3).
Code staleness: gen.go is stale (v2 vs v3).
gen_test.go is up to date (v1 matches).

Expect exit code 1. All three sections have entries.

## Operational Error

### code-from-spec directory missing

Create a TempDir with no `code-from-spec/` subdirectory.

Expect exit code 2 and stderr containing an error
message.

## Output Ordering

### Nodes sorted alphabetically

Create:
- `code-from-spec/_node.md`: version=2, title=`ROOT`
- `code-from-spec/domain/_node.md`: version=1,
  parent_version=1, title=`ROOT/domain`
- `code-from-spec/arch/_node.md`: version=1,
  parent_version=1, title=`ROOT/arch`

Both have parent_changed (1 vs 2).

Expect spec_staleness entries in order: `ROOT/arch`
before `ROOT/domain`.
