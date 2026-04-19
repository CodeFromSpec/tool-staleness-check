# External Dependencies

External dependencies represent knowledge that lives outside the
project — third-party APIs, shared libraries, databases owned by other
teams.

---

## Location

Each external dependency is a folder under `external/`.

---

## Logical names

The logical name of an external dependency is `EXTERNAL/` followed by the
folder name — e.g., folder `database/` becomes `EXTERNAL/database`.

---

## Contents

### _external.md

Every dependency folder must contain an `_external.md` at its root. It
is the entry point — the only file with a controlled format.

Frontmatter contains a single field:

```yaml
---
version: 1
---
```

See `VERSIONING_AND_STALENESS.md` for when to increment `version`.
External dependencies do not support `depends_on` by design — they
are independent of each other and of the spec tree.

The body starts with a title using the dependency's logical name:

```markdown
# EXTERNAL/dependency-name
```

### Other files

The folder may contain any supporting files and subfolders — imported
artifacts, API docs, reference material — organized freely.

---

## Referencing

Externals are referenced via `depends_on`, using the dependency's
logical name as path:

```yaml
depends_on:
  - path: EXTERNAL/database
    version: 5
```

This imports the contents of the dependency's `_external.md` and all
files in its folder.

When only a subset of files is needed, use `filter` — an array of glob
patterns relative to the dependency folder:

```yaml
depends_on:
  - path: EXTERNAL/celcoin-api
    version: 5
    filter:
      - "api/onboarding-create-pf*"
      - "reference/security*"
```

Filters are additive — a file matching any pattern is imported. The
`_external.md` is always imported regardless of filter.
