---
version: 7
parent_version: 10
---

# ROOT/domain/staleness

## Intent

Defines versioning rules and staleness conditions.

## Contracts

### Which files are versioned

| File | Location |
|---|---|
| Spec node | `code-from-spec/**/_node.md` |
| Test node | `code-from-spec/**/*.test.md` |

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
| Test node (`*.test.md`) | Subject or dependency version changed. The subject is the `_node.md` in the same directory. |
| Generated source file | Node version changed since last generation |

### How to determine if a spec node is stale

A spec node is stale when:

```
parent.version != node.parent_version
depends_on[x].current_version != node.depends_on[x].version
```

### How to determine if a test node is stale

A test node is stale when:

```
subject.version != node.subject_version
depends_on[x].current_version != node.depends_on[x].version
```

The subject is the `_node.md` in the same directory as
the test node.

### How to determine if a generated file is stale

A generated source file is stale when:

```
node.version != version in the file's spec comment
```
