# Agent: Code Staleness

Rules for code staleness verification.

---

## Input

This file is provided to you as instructions before any other
content. After these instructions, you receive a YAML block
listing all nodes to check:

```yaml
nodes:
  - code-from-spec/spec/architecture/backend/config/_node.md
  - code-from-spec/spec/architecture/backend/config/default.test.md
  - code-from-spec/spec/architecture/backend/api/api-router/_node.md
  - code-from-spec/spec/architecture/backend/types/_node.md
```

---

## Output

Report a status for every node in the input and every file in its
`implements`.

Possible statuses:

| Status | Meaning |
|---|---|
| `ok` | Versions match. File is up to date. |
| `stale` | Versions differ. |
| `missing` | File in `implements` does not exist. |
| `no_spec_comment` | File exists but has no spec comment. |
| `malformed_spec_comment` | Spec comment exists but cannot be parsed. |
| `wrong_node` | Spec comment references a different node than expected. |
| `node_not_found` | Input node file does not exist. |
| `unreadable_frontmatter` | Node file exists but frontmatter cannot be parsed or does not exist. |
| `no_version` | Frontmatter exists but has no `version` field. |
| `no_implements` | Frontmatter exists but has no `implements` field. |
| `empty_implements` | `implements` field exists but is an empty list. |

Every entry includes `node` and `status`. The `file` field is
included when the status refers to a specific generated file.
Every input node must produce at least one entry in the output.
Do not skip any node or file.

Produce a YAML list:

```yaml
results:
  - node: code-from-spec/spec/architecture/backend/config/_node.md
    file: internal/configuration/config.go
    status: ok
  - node: code-from-spec/spec/architecture/backend/config/default.test.md
    file: internal/configuration/config_test.go
    status: stale
  - node: code-from-spec/spec/architecture/backend/api/api-router/_node.md
    status: no_implements
  - node: code-from-spec/spec/architecture/backend/types/_node.md
    file: internal/types/types.h
    status: ok
  - node: code-from-spec/spec/architecture/backend/types/_node.md
    file: internal/types/types.c
    status: stale
```

---

## Principles

- **Read only what is needed.** From each node, read only the
  frontmatter to get `version` and `implements`. For generated
  files, use a search (e.g., grep, including via bash if needed)
  to find the spec comment rather than reading the full file.
- **Report, do not fix.** Your job is to identify and report
  staleness. Do not modify any files. Do not regenerate code.
- **No other operations.** Do not fetch any information beyond
  node frontmatters and spec comments in generated files.

---

## Definitions

The **spec comment** uses the language's single-line comment syntax
followed by `spec: <logical-name>@v<version>`. Examples:
`// spec: ROOT/path@v3` (Go, Java, C),
`# spec: ROOT/path@v3` (Python, Ruby),
`// spec: TEST/path@v3` (test node, canonical),
`// spec: TEST/path(edge_cases)@v3` (test node, named).

Logical name derivation from node path:
- Spec node `code-from-spec/spec/x/_node.md` → `ROOT/x`
- Test node `code-from-spec/spec/x/y.test.md` → `TEST/x(y)`
- Test node `code-from-spec/spec/x/default.test.md` → `TEST/x`
  (alias of `TEST/x(default)`; both forms are valid in spec
  comments — either must match)

A generated source file is stale when:
`node.version != version in the file's spec comment`

---

## Procedure

For each `_node.md` path in the input:

1. If the node file does not exist, add an entry with status
   `node_not_found` and skip to the next node.
2. Try to read the node's frontmatter. If it cannot be parsed,
   add an entry with status `unreadable_frontmatter` and skip to
   the next node.
3. Look for the `version` field in the frontmatter. If missing,
   add an entry with status `no_version` and skip to the next
   node.
4. Look for the `implements` field in the frontmatter (a YAML
   list of file paths).
   - If missing, add an entry with status `no_implements` and
     skip to the next node.
   - If present but empty, add an entry with status
     `empty_implements` and skip to the next node.
5. For each file in `implements`, add an entry to the output:
   - If the file does not exist, set status to `missing`.
   - If the file exists but has no spec comment, set status to
     `no_spec_comment`.
   - If the spec comment exists but cannot be parsed (e.g.,
     missing version, malformed format), set status to
     `malformed_spec_comment`.
   - If the spec comment references a different node than the one
     being checked, set status to `wrong_node`.
   - If versions match, set status to `ok`.
   - If versions differ, set status to `stale`.
