---
version: 14
parent_version: 11
depends_on:
  - path: ROOT/domain/output
    version: 7
  - path: ROOT/tech_design/internal/code_staleness
    version: 9
  - path: ROOT/tech_design/internal/discovery
    version: 13
  - path: ROOT/tech_design/internal/frontmatter
    version: 10
  - path: ROOT/tech_design/internal/spec_staleness
    version: 10
implements:
  - cmd/staleness-check/main.go
---

# ROOT/tech_design/main

## Intent

Entry point that orchestrates discovery, parsing,
verification, and output.

## Contracts

### Arguments

If any argument is passed (e.g., `--help`), print the
help message below to stdout and exit 0. Otherwise
proceed with verification.

### Help message

```
staleness-check — verifies spec and code staleness for a Code from Spec project.

Usage: staleness-check

Run from the project root with no arguments.
Outputs YAML to stdout with three sections:

  spec_staleness:
    - node: <logical-name>
      statuses:
        - <status>
  test_staleness:
    - node: <logical-name>
      statuses:
        - <status>
  code_staleness:
    - node: <logical-name>
      file: <file-path>
      status: <status>

Sections with no problems are empty lists ([]).

Spec and test staleness statuses:
  invalid_frontmatter  Frontmatter cannot be parsed or is missing required fields.
  wrong_name           Title does not match expected logical name.
  invalid_parent       Parent file cannot be found or read. (spec nodes)
  parent_changed       Parent version changed. (spec nodes)
  invalid_subject      Subject file cannot be found or read. (test nodes)
  subject_changed      Subject version changed. (test nodes)
  invalid_dependency   Dependency is malformed or cannot be found or read.
  dependency_changed   Dependency version changed.

Code staleness statuses:
  unreadable_frontmatter  Frontmatter cannot be parsed.
  no_version              Frontmatter has no version field.
  missing                 File in implements does not exist.
  no_spec_comment         File exists but has no spec comment.
  malformed_spec_comment  Spec comment exists but cannot be parsed.
  wrong_node              Spec comment references a different node.
  stale                   Spec version differs from spec comment version.

Exit codes: 0 = no problems, 1 = problems found, 2 = operational error.
```

### Execution flow

1. Call `DiscoverNodes` to find all spec nodes and test
   nodes.
2. Build the frontmatter cache: call `ParseFrontmatter`
   for every discovered node (spec and test).
   Store the result in a `map[string]*Frontmatter` keyed
   by file path. On success, store the `*Frontmatter`.
   On failure, store `nil` — do not abort.
3. Run spec staleness: call `CheckSpecStaleness` for
   each spec node, sorted alphabetically by logical name.
   Collect all results.
4. Run test staleness: call `CheckSpecStaleness` for
   each test node, sorted alphabetically by logical name.
   Collect all results.
5. Run code staleness: call `CheckCodeStaleness` for
   each spec and test node, sorted alphabetically by
   logical name. Collect all results.
6. Emit YAML to stdout with three sections:
   `spec_staleness`, `test_staleness`, `code_staleness`.
7. Exit with code 0 if all sections are empty, 1 if any
   section has entries, 2 on operational error.

### Output format

YAML document to stdout. Three top-level keys in order.
Spec and test staleness entries use `statuses` (list).
Code staleness entries use `status` (string).

Only nodes/files with problems are included. Empty
sections are `[]`. See `ROOT/domain/output` for full
format specification and examples.

### Operational errors

If `DiscoverNodes` fails, print the error to stderr and
exit 2. Frontmatter parse failures are not operational
errors — they are captured as `nil` in the cache and
surfaced as statuses during verification.

### YAML serialization

Use `github.com/goccy/go-yaml` for output. The output struct
must produce the exact field names and format prescribed
by `ROOT/domain/output`.
