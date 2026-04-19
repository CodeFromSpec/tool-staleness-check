---
version: 4
parent_version: 10
---

# ROOT/domain/external_dependencies

## Intent

Defines the structure of external dependencies as relevant
to staleness verification.

## Contracts

### Location

Each external dependency is a folder under `external/`.
The `external/` directory itself is optional — a project
may have no external dependencies.

### Logical names

The logical name of an external dependency is `EXTERNAL/`
followed by the folder name — e.g., folder `database/`
becomes `EXTERNAL/database`.

### Contents

Every dependency folder must contain an `_external.md` at
its root. Frontmatter contains at least `version`:

```yaml
---
version: 1
---
```

Other frontmatter fields may exist but are not relevant
for staleness verification.

The folder may contain other files and subfolders. The
tool must ignore them — only `_external.md` is read.
