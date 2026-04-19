# Agent: Spec Staleness

Rules for spec staleness verification.

---

## Input

This file is provided to you as instructions before any other
content. After these instructions, you receive a YAML block
listing all nodes to check, using logical names:

```yaml
nodes:
  - ROOT
  - ROOT/architecture
  - ROOT/architecture/backend
  - ROOT/architecture/backend/config
  - TEST/architecture/backend/config
  - TEST/architecture/backend/config(edge_cases)
```

The input contains everything you need. Do not search for or
discover additional files.

---

## Output

Report a status for every node in the input.

Possible statuses:

| Status | Meaning |
|---|---|
| `ok` | All declared versions match current versions. |
| `node_not_found` | Input node file does not exist. |
| `invalid_frontmatter` | Node frontmatter is unreadable or missing required fields. |
| `invalid_parent` | Parent node cannot be read or does not exist. |
| `stale_parent` | `parent_version` does not match parent's `version`. |
| `invalid_dependency` | A `depends_on` entry is malformed or the referenced file cannot be read. |
| `stale_dependency` | A `depends_on` version does not match the dependency's current `version`. |

Report the first problem found for each node and move to the next.
A clean node produces one entry with status `ok`.

Every entry includes `node` and `status`. Every input node must
produce exactly one entry in the output. Do not skip any node.

Produce a YAML list:

```yaml
results:
  - node: ROOT
    status: ok
  - node: ROOT/architecture
    status: ok
  - node: ROOT/architecture/backend
    status: stale_parent
  - node: ROOT/architecture/backend/config
    status: stale_dependency
  - node: TEST/architecture/backend/config
    status: stale_parent
```

---

## Principles

- **Read only what is needed.** From each file, read only the
  frontmatter. Do not read full file bodies.
- **Use only the Read tool.** Do not use search, glob, grep, bash,
  or any other tool. The only filesystem operation allowed is
  reading files via the Read tool.
- **Read only allowed files.** You may only read a file if it was
  listed in the input, or if its path was discovered in the
  frontmatter of a file you already read (parent, dependency, or
  test subject). No other files may be read for any reason.
  Reading any file outside this chain of trust is an error. You
  are being evaluated on your ability to not read unauthorized
  files.
- **Cache reads.** Read each file at most once. If you already read
  a file's frontmatter (e.g., as a node in the input), reuse that
  result when the same file appears as a parent or dependency.
- **Report, do not fix.** Your job is to identify and report
  staleness. Do not modify any files. Do not resolve staleness.

---

## Definitions

The root node is `ROOT`. It has no parent. Every other node is a
non-root node.

The parent of a spec node is determined by its logical name —
`ROOT/architecture/backend/config` has parent
`ROOT/architecture/backend`.

The parent of a test node is the `_node.md` in the same directory —
`TEST/architecture/backend/config(edge_cases)` has parent
`ROOT/architecture/backend/config`.

Logical name resolution:
- `ROOT/x` → `code-from-spec/spec/x/_node.md`
- `TEST/x(y)` → `code-from-spec/spec/x/y.test.md`
- `TEST/x` is an alias for `TEST/x(default)`
- `EXTERNAL/x` → `code-from-spec/external/x/_external.md`

---

## Procedure

First, read the frontmatter of every node in the input and cache
the results. Then process each node:

1. If the node file does not exist, add an entry with status
   `node_not_found` and skip to the next node.
2. If the frontmatter could not be parsed or is missing required
   fields (`version`, and `parent_version` for non-root nodes),
   add an entry with status `invalid_frontmatter` and skip to
   the next node.
3. If the node is not root, locate the parent node. Use the
   cached frontmatter if available, otherwise read it.
   - If the parent cannot be found or its frontmatter cannot be
     read, add an entry with status `invalid_parent` and skip to
     the next node.
   - Otherwise, compare the `version` field in the parent's
     frontmatter with the node's `parent_version`. If they differ,
     add an entry with status `stale_parent` and skip to the next
     node.
4. For each `depends_on` entry, resolve the logical name. Use the
   cached frontmatter if available, otherwise read it.
   - If the entry is malformed or the referenced file cannot be
     found or read, add an entry with status
     `invalid_dependency` and skip to the next node.
   - Otherwise, compare the `version` field in the dependency's
     frontmatter with the `version` declared in the `depends_on`
     entry. If they differ, add an entry with status
     `stale_dependency` and skip to the next node.
5. If none of the above produced an entry, add an entry with
   status `ok`.
