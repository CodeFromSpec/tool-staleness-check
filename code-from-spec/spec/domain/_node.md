---
version: 10
parent_version: 9
---

# ROOT/domain

## Intent

Defines the domain rules for the staleness verification
tool. The tool performs both spec staleness and code
staleness verification as defined by the Code from Spec
methodology.

## Context

### The model

Specifications are organized as a tree. Each node adds
precision to its parent — business intent at the root,
implementation contracts at the leaves. Only leaf nodes
generate code.

```
root/
└── payments/
    └── fees/
        ├── calculation/   ← leaf, implemented
        └── rounding/      ← leaf, implemented
```

### Structure

A Code from Spec project contains:

```
project_root/
  code-from-spec/
    spec/       ← spec tree
    external/   ← external dependencies
```

### Spec vs. code staleness

Spec staleness and code staleness are conceptually
separate. Spec staleness detects nodes whose parent or
dependencies changed. Code staleness detects generated
files that are out of sync with their spec. Spec staleness
must be resolved before code staleness results are
meaningful — the tool reports both but in separate sections
to make this dependency clear.
