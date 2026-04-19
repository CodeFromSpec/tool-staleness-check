# Versioning and Staleness

Every versioned file has a `version` field in its YAML frontmatter.
Version numbers are integers.

---

## Which files are versioned

| File | Location |
|---|---|
| Spec node | `spec/**/_node.md` |
| Test node | `spec/**/*.test.md` |
| External dependency | `external/**/_external.md` |

---

## When to increment the version field

The `version` field must be incremented on every change to the
file — no exceptions. A single added space, a corrected typo, a
reformatted line, a bumped dependency version in the frontmatter —
all require a version increment. The rule is mechanical: if
computing a hash of the file before and after the change would
produce different results, the version must change. Semantic
significance is irrelevant. Never decide that a change is "too
small" to warrant a version increment.

For external dependencies, the rule extends beyond the
`_external.md` itself: any change to any file in the dependency
folder requires incrementing the version in `_external.md`.

---

## How to increment

Add 1 to the current value. Version 3 becomes 4, not 5 or 10.

---

## What is staleness

A file is stale when it references a version that is no longer
current — meaning something it depends on has changed since it was
last updated. Staleness is never declared — it is always
calculated by comparing declared versions against current versions.

---

## Which files can become stale

| File | Stale when |
|---|---|
| Spec node (`_node.md`) | Parent or dependency version changed |
| Test node (`*.test.md`) | Parent or dependency version changed. The parent of a test node is the `_node.md` in the same directory. |
| Generated source file | Node version changed since last generation |

External dependencies do not become stale — they are external
sources of truth. When they change, the nodes that depend on them
become stale.

---

## How to determine if a file is stale

A node is stale when:

```
parent.version != node.parent_version
depends_on[x].current_version != node.depends_on[x].version
```

For test nodes, the parent is the `_node.md` in the same directory.

A generated source file is stale when:

```
node.version != version in the file's // spec: comment
```
