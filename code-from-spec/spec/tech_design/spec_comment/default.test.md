---
version: 1
parent_version: 6
implements:
  - cmd/staleness-check/speccomment_test.go
---

# TEST/tech_design/spec_comment

## Context

Each test uses `t.TempDir()` to create an isolated
temporary directory. Test files are created with
controlled content. `ParseSpecComment` is called with
the path to each test file.

## Happy Path

### Go-style comment

Create a file with:

```
// spec: ROOT/architecture/backend/config@v5
package configuration
```

Expect `LogicalName` = `"ROOT/architecture/backend/config"`,
`Version` = 5.

### Python-style comment

Create a file with:

```
# spec: ROOT/domain/staleness@v3
```

Expect `LogicalName` = `"ROOT/domain/staleness"`,
`Version` = 3.

### HTML-style comment

Create a file with:

```
<!-- spec: ROOT/frontend/template@v2 -->
```

Expect `LogicalName` = `"ROOT/frontend/template"`,
`Version` = 2.

### Block comment single line

Create a file with:

```
/* spec: ROOT/some/node@v1 */
```

Expect `LogicalName` = `"ROOT/some/node"`,
`Version` = 1.

### Test node canonical

Create a file with:

```
// spec: TEST/domain/config@v3
```

Expect `LogicalName` = `"TEST/domain/config"`,
`Version` = 3.

### Test node named

Create a file with:

```
// spec: TEST/domain/config(edge_cases)@v2
```

Expect `LogicalName` =
`"TEST/domain/config(edge_cases)"`, `Version` = 2.

### Comment not on first line

Create a file with a shebang and license before the
spec comment:

```
#!/usr/bin/env python3
# License: MIT
# spec: ROOT/scripts/deploy@v1
```

Expect `LogicalName` = `"ROOT/scripts/deploy"`,
`Version` = 1.

### SQL-style comment

Create a file with:

```
-- spec: ROOT/database/migrations@v4
```

Expect `LogicalName` = `"ROOT/database/migrations"`,
`Version` = 4.

## Edge Cases

### Spec comment deep in file

Create a file with 100 lines of code before the spec
comment. Expect it to still be found.

### Trailing whitespace after version

Create a file with:

```
// spec: ROOT/node@v3   
```

Expect `LogicalName` = `"ROOT/node"`, `Version` = 3.

### Logical name with many segments

Create a file with:

```
// spec: ROOT/a/b/c/d/e/f@v1
```

Expect `LogicalName` = `"ROOT/a/b/c/d/e/f"`,
`Version` = 1.

## Failure Cases

### File does not exist

Call `ParseSpecComment` with a non-existent path.
Expect an error containing the file path.

### No spec comment in file

Create a file with no `spec:` substring:

```
package main

func main() {}
```

Expect an error indicating no spec comment found.

### Missing version

Create a file with:

```
// spec: ROOT/node@v
```

Expect an error indicating malformed spec comment.

### Missing @v separator

Create a file with:

```
// spec: ROOT/node
```

Expect an error indicating malformed spec comment.

### Non-integer version

Create a file with:

```
// spec: ROOT/node@vabc
```

Expect an error indicating malformed spec comment.

### Empty logical name

Create a file with:

```
// spec: @v3
```

Expect an error indicating malformed spec comment.
