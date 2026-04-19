---
version: 6
parent_version: 10
---

# ROOT/domain/staleness

## Intent

Defines versioning rules and staleness conditions.

## Contracts

### Which files are versioned

| File | Location |
|---|---|
| Spec node | `spec/**/_node.md` |
| Test node | `spec/**/*.test.md` |
| External dependency | `external/**/_external.md` |

### What is staleness

A file is stale when it references a version that is no
longer current — meaning something it depends on has
changed since it was last updated. Staleness is never
declared — it is always calculated by comparing declared
versions against current versions.

### Which files can become stale

| File | Stale when |
|---|---|
| Spec node (`_node.md`) | Parent or dependency version changed |
| Test node (`*.test.md`) | Parent or dependency version changed. The parent of a test node is the `_node.md` in the same directory. |
| Generated source file | Node version changed since last generation |

External dependencies do not become stale — they are
external sources of truth. When they change, the nodes
that depend on them become stale.

### How to determine if a spec node is stale

A node is stale when:

```
parent.version != node.parent_version
depends_on[x].current_version != node.depends_on[x].version
```

For test nodes, the parent is the `_node.md` in the same
directory.


### How to determine if a generated file is stale

A generated source file is stale when:

```
node.version != version in the file's spec comment
```
