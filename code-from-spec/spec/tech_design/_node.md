---
version: 14
parent_version: 9
---

# ROOT/tech_design

## Intent

Technical design decisions for implementing the staleness
verification tool in Go.

## Context

This is a single-purpose CLI tool — no server, no library
API, no plugin system. The design prioritizes simplicity,
correctness, and fast execution.

## Contracts

### Language

Go (minimum 1.22).

### Go module

The module path is:
`github.com/CodeFromSpec/tool-staleness-check/v2`

### Dependencies

- Standard library only, plus `github.com/goccy/go-yaml` for YAML
  parsing and output.
- No test framework beyond the standard `testing` package.

### Error handling

- **Operational errors** (cannot read `spec/` directory,
  permission denied, invalid working directory) — print
  to stderr, exit 2. Error messages must be clear and
  actionable — they should tell the user what went wrong
  and how to fix it.
- **Node-level problems** (missing file, bad frontmatter,
  unresolvable parent) — captured as a status in the
  result, not as an operational error. The tool continues
  to the next node.
- The tool never panics. All errors are handled.
- Every error return value must be checked. No
  unhandled errors — the code must pass linters that
  enforce this (e.g., `errcheck`).

## Constraints

- The tool reads frontmatter and the title line from node
  files — never the rest of the body.
- For generated files (code staleness), the tool reads
  only enough to find the spec comment — not the full
  file.
- A file's frontmatter is read at most once and cached
  for reuse when the same file appears as a parent or
  dependency.
- No global state. All state is passed explicitly.
- No concurrency. The tool is fast enough single-threaded
  for any realistic spec tree.
- No configuration files. Behavior is fully determined by
  the spec tree on disk.

## Test Conventions

- All test helper functions and types must be prefixed with `test`
  (e.g., `testMakeFM`, `testIntPtr`, `testCase`). This prevents name
  collisions with unexported functions and types in the package under
  test when using internal test files (same package as the
  implementation).
