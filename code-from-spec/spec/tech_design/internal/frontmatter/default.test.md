---
version: 7
parent_version: 12
implements:
  - internal/frontmatter/frontmatter_test.go
---

# TEST/tech_design/internal/frontmatter

## Context

Each test uses `t.TempDir()` to create an isolated
temporary directory. Test files are created with
controlled frontmatter content. `ParseFrontmatter` is
called with the path to each test file.

## Happy Path

### Parses complete frontmatter

Create a file with all fields:

```
---
version: 3
parent_version: 2
depends_on:
  - path: ROOT/other
    version: 1
  - path: ROOT/another
    version: 5
implements:
  - internal/config/config.go
  - internal/config/config_test.go
---

# ROOT/some/node
```

Expect all fields populated correctly, including
`Title` = `"ROOT/some/node"`.

### Parses root node (no parent_version)

Create a file with only `version`:

```
---
version: 5
---

# ROOT
```

Expect `Version` = 5, `ParentVersion` = nil,
`SubjectVersion` = nil, `DependsOn` = nil,
`Implements` = nil, `Title` = `"ROOT"`.

### Parses test node with subject_version

Create a file with `subject_version`:

```
---
version: 2
subject_version: 5
implements:
  - internal/config/config_test.go
---

# TEST/some/node
```

Expect `Version` = 2, `SubjectVersion` pointing to 5,
`ParentVersion` = nil, `Title` = `"TEST/some/node"`.

### Parses external dependency

Create a file with only `version`:

```
---
version: 2
---

# ROOT/external/database
```

Expect `Version` = 2, `Title` = `"ROOT/external/database"`.

### Ignores unknown frontmatter fields

Create a file with extra fields:

```
---
version: 1
parent_version: 1
some_future_field: hello
another: 42
---

# ROOT/node
```

Expect no error. Known fields parsed correctly.
Unknown fields ignored.

### Parses test node title

Create a file with:

```
---
version: 1
subject_version: 2
implements:
  - internal/config/config_test.go
---

# TEST/some/node
```

Expect `Title` = `"TEST/some/node"`.

### Parses named test node title

Create a file with:

```
---
version: 1
subject_version: 2
implements:
  - internal/config/config_edge_test.go
---

# TEST/some/node(edge_cases)
```

Expect `Title` = `"TEST/some/node(edge_cases)"`.

## Edge Cases

### Empty frontmatter

Create a file with:

```
---
---

# ROOT/node
```

Expect `Version` = nil, all other fields zero/nil,
`Title` = `"ROOT/node"`. No error.

### Title with blank lines between frontmatter and title

Create a file with blank lines after the closing `---`:

```
---
version: 1
---


# ROOT/node
```

Expect `Title` = `"ROOT/node"`. Blank lines are
skipped when searching for the title.

### No title line

Create a file with frontmatter but no `# ` line after:

```
---
version: 1
---

Some text without a title.
```

Expect `Title` = `""`. No error.

### File with only frontmatter, nothing after

Create a file with:

```
---
version: 1
---
```

Expect `Title` = `""`. No error.

## Failure Cases

### File does not exist

Call `ParseFrontmatter` with a non-existent path.
Expect an error containing the file path.

### No frontmatter delimiters

Create a file with no `---` at all:

```
Just some text.
```

Expect an error indicating frontmatter not found.

### Malformed YAML in frontmatter

Create a file with invalid YAML between delimiters:

```
---
version: [invalid
---
```

Expect an error indicating parse failure.
