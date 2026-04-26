# staleness-check

Command-line tool that verifies spec and code staleness for a [Code from Spec](code-from-spec/CODE_FROM_SPEC.md) project.

Code from Spec originally delegated staleness verification to AI agents. In practice, agent-based verification is expensive in tokens and unreliable — LLMs can misread frontmatter, skip nodes, or hallucinate statuses. Since the verification logic is fully deterministic (read frontmatter, compare version numbers, report results), a compiled tool eliminates the cost and reliability problems while producing identical results.

## Installation

Download the binary for your platform from the [latest release](https://github.com/CodeFromSpec/tool-staleness-check/releases/latest) and place it somewhere on your `PATH`.

### Build from source

Requires Go 1.24+.

```bash
go build -o staleness-check ./cmd/staleness-check
```

## Usage

Run from the project root with no arguments:

```bash
./staleness-check
```

Pass any argument to see the help message:

```bash
./staleness-check --help
```

## Output

YAML to stdout with three sections:

```yaml
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
```

Sections with no problems are empty lists (`[]`).

### Spec and test staleness statuses

| Status | Meaning |
|---|---|
| `invalid_frontmatter` | Frontmatter cannot be parsed or is missing required fields. |
| `wrong_name` | Title does not match expected logical name. |
| `invalid_parent` | Parent file cannot be found or read. (spec nodes) |
| `parent_changed` | Parent version changed. (spec nodes) |
| `invalid_subject` | Subject file cannot be found or read. (test nodes) |
| `subject_changed` | Subject version changed. (test nodes) |
| `invalid_dependency` | Dependency is malformed or cannot be found or read. |
| `dependency_changed` | Dependency version changed. |

### Code staleness statuses

| Status | Meaning |
|---|---|
| `unreadable_frontmatter` | Frontmatter cannot be parsed. |
| `no_version` | Frontmatter has no version field. |
| `missing` | File in implements does not exist. |
| `no_spec_comment` | File exists but has no spec comment. |
| `malformed_spec_comment` | Spec comment exists but cannot be parsed. |
| `wrong_node` | Spec comment references a different node. |
| `stale` | Spec version differs from spec comment version. |

## Exit codes

| Code | Meaning |
|---|---|
| `0` | No problems found. |
| `1` | Problems found. |
| `2` | Operational error. |

## Tests

```bash
go test ./...
```

## Project structure

```
cmd/staleness-check/     Entry point (main.go)
internal/                Internal packages (codestaleness, discovery, frontmatter,
                         logicalnames, speccomment, specstaleness)
code-from-spec/spec/     Specification tree (source of truth)
code-from-spec/framework/ Code from Spec framework docs
```

The specification tree under `code-from-spec/spec/` is the authoritative source for every behavior of this tool. Each Go file references the spec node it implements via a spec comment on its first line.

## Versioning

The major version of this tool tracks the version of the [Code from Spec](code-from-spec/CODE_FROM_SPEC.md) methodology it supports — not semver compatibility. `v2.x.y` supports Code from Spec v2; a future `v3.x.y` would support Code from Spec v3.
