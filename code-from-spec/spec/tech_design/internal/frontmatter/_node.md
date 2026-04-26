---
version: 10
parent_version: 1
depends_on:
  - path: ROOT/domain/specifications
    version: 4
  - path: ROOT/domain/name_verification
    version: 3
implements:
  - internal/frontmatter/frontmatter.go
---

# ROOT/tech_design/internal/frontmatter

## Intent

Reads and parses the YAML frontmatter and title from node
files.

## Contracts

### Package

`frontmatter`

### Parsing

The frontmatter is the YAML block between the first `---`
and the second `---` at the top of the file. Everything
after the second `---` is ignored except for the title.

Fields extracted from frontmatter:
- `version` (pointer to integer, nil if absent)
- `parent_version` (pointer to integer, nil if absent)
- `subject_version` (pointer to integer, nil if absent)
- `depends_on` (list of objects with `path` string and
  `version` integer)
- `implements` (list of strings)

All fields are optional at the parsing level — validation
of required fields happens during staleness verification.
Unknown fields are ignored.

### Title extraction

The title is the first non-empty line after the
frontmatter closing `---`. It is expected to start with
`# ` followed by the logical name. The parser extracts
the text after `# ` and stores it alongside the
frontmatter fields.

If the title line is missing or does not start with `# `,
the title is stored as empty string — the caller decides
how to handle it.

### Interface

```go
type DependsOn struct {
    Path    string
    Version int
}

type Frontmatter struct {
    Version        *int
    ParentVersion  *int
    SubjectVersion *int
    DependsOn      []DependsOn
    Implements     []string
    Title          string
}

func ParseFrontmatter(filePath string) (
    *Frontmatter, error,
)
```

`ParseFrontmatter` reads the file, extracts the
frontmatter and title, and returns the result. It does
not cache — caching is the caller's responsibility.

### Efficiency

The parser must not read the entire file into memory.
It reads line by line, extracts the frontmatter and
title, and stops as soon as it has both — the rest of
the file is never read. Intermediate state (e.g., the
raw frontmatter lines) is discarded after parsing —
only the final `Frontmatter` struct is retained. This
matters because spec files can have long bodies that
are irrelevant to staleness verification.

### Error handling

Errors must wrap the underlying error with a descriptive
message:
- `error reading <path>: <underlying error>`
- `error parsing frontmatter in <path>: <underlying error>`
- `frontmatter not found in <path>`
