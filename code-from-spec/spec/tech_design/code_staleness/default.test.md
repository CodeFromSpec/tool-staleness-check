---
version: 1
parent_version: 2
implements:
  - cmd/staleness-check/codestaleness_test.go
---

# TEST/tech_design/code_staleness

## Context

Tests build a `cache` (`map[string]*Frontmatter`) with
controlled entries. Generated files are created in
`t.TempDir()` with controlled content. `ParseSpecComment`
is called on real files — no mocking. `Version` in
`Frontmatter` is `*int` — use helper to create pointer
values.

## Happy Path

### All files up to date

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  Implements=[`<tmpdir>/config.go`]

Create `<tmpdir>/config.go` with content:
```
// spec: ROOT/domain/config@v2
package config
```

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: empty slice.

### Multiple files all up to date

Cache contains:
- `spec/domain/config/_node.md`: Version=3,
  Implements=[`<tmpdir>/config.go`, `<tmpdir>/util.go`]

Create both files with `// spec: ROOT/domain/config@v3`.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: empty slice.

### Test node with canonical equivalence

Cache contains:
- `spec/domain/config/default.test.md`: Version=1,
  Implements=[`<tmpdir>/config_test.go`]

Create `<tmpdir>/config_test.go` with:
```
// spec: TEST/domain/config(default)@v1
package config
```

Node: LogicalName=`"TEST/domain/config"`,
FilePath=`"spec/domain/config/default.test.md"`.

Expect: empty slice (LogicalNamesMatch treats
`TEST/domain/config` and `TEST/domain/config(default)`
as equal).

### No implements

Cache contains:
- `spec/domain/_node.md`: Version=5, Implements=nil

Node: LogicalName=`"ROOT/domain"`,
FilePath=`"spec/domain/_node.md"`.

Expect: empty slice.

## Blocking Steps (1-2)

### Node not in cache

Cache is empty.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with Node=`"ROOT/domain/config"`,
File=`""`, Status=`"unreadable_frontmatter"`.

### Node nil in cache

Cache contains:
- `spec/domain/config/_node.md`: nil

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with
Status=`"unreadable_frontmatter"`.

### Version nil

Cache contains:
- `spec/domain/config/_node.md`: Version=nil,
  Implements=[`<tmpdir>/config.go`]

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with Status=`"no_version"`.

## Per-file Statuses

### missing — file does not exist

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  Implements=[`<tmpdir>/nonexistent.go`]

Do not create the file.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with
Node=`"ROOT/domain/config"`,
File=`"<tmpdir>/nonexistent.go"`,
Status=`"missing"`.

### no_spec_comment

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  Implements=[`<tmpdir>/config.go`]

Create `<tmpdir>/config.go` with:
```
package config

func Init() {}
```

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with Status=`"no_spec_comment"`.

### malformed_spec_comment

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  Implements=[`<tmpdir>/config.go`]

Create `<tmpdir>/config.go` with:
```
// spec: ROOT/domain/config@vabc
package config
```

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with
Status=`"malformed_spec_comment"`.

### wrong_node

Cache contains:
- `spec/domain/config/_node.md`: Version=2,
  Implements=[`<tmpdir>/config.go`]

Create `<tmpdir>/config.go` with:
```
// spec: ROOT/domain/other@v2
package config
```

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: single result with Status=`"wrong_node"`.

### stale

Cache contains:
- `spec/domain/config/_node.md`: Version=3,
  Implements=[`<tmpdir>/config.go`]

Create `<tmpdir>/config.go` with:
```
// spec: ROOT/domain/config@v2
package config
```

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Version is 3 but spec comment says v2.

Expect: single result with Status=`"stale"`.

## Multiple Files

### Mixed results across files

Cache contains:
- `spec/domain/config/_node.md`: Version=3,
  Implements=[`<tmpdir>/a.go`, `<tmpdir>/b.go`,
  `<tmpdir>/c.go`]

Create `<tmpdir>/a.go` with:
```
// spec: ROOT/domain/config@v3
package config
```
Create `<tmpdir>/b.go` with:
```
// spec: ROOT/domain/config@v2
package config
```
Do not create `<tmpdir>/c.go`.

Node: LogicalName=`"ROOT/domain/config"`,
FilePath=`"spec/domain/config/_node.md"`.

Expect: two results:
- File=`"<tmpdir>/b.go"`, Status=`"stale"`
- File=`"<tmpdir>/c.go"`, Status=`"missing"`

(a.go is up to date, omitted from results.)
