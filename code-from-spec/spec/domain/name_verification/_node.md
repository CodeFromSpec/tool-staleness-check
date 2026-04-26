---
version: 3
parent_version: 10
---

# ROOT/domain/name_verification

## Intent

Defines the rule for verifying that a node's title
matches its position in the filesystem.

## Context

If a node is moved from one parent to another, and both
parents happen to have the same version number, the
`parent_version` check passes even though the node is
under a different parent. Comparing the node's declared
title against the logical name derived from its filesystem
path detects this case.

This verification is specific to this tool — it is not
part of the original Code from Spec staleness procedures.

## Contracts

### Title location

The title is the first non-empty line after the
frontmatter closing `---`. It uses the format:

```
# <logical-name>
```

Examples:
- `# ROOT`
- `# ROOT/domain/staleness`
- `# TEST/architecture/backend/config`

### Verification rule

The logical name in the title must match the logical name
derived from the node's filesystem path. If they do not
match, the node fails name verification.

This check applies to all discovered nodes (spec nodes
and test nodes) and is performed as part of spec
staleness verification.
