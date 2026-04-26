---
version: 11
parent_version: 2
depends_on:
  - path: ROOT/domain/specifications
    version: 4
implements:
  - internal/speccomment/speccomment.go
---

# ROOT/tech_design/internal/spec_comment

## Intent

Extracts the spec reference comment from generated source
files for code staleness verification.

## Contracts

### Package

`speccomment`

### Pattern

The spec comment contains the substring:

```
code-from-spec: <logical-name>@v<version>
```

The tool does not attempt to identify the comment syntax
of the language. It scans each line for the pattern
regardless of what precedes or follows it. This makes it
language-agnostic — any comment syntax works.

### Detection

Read the file line by line from the top. For each line,
look for the substring `code-from-spec: `. Stop reading
as soon as a match is found. If the entire file is read
without a match, report that no spec comment was found.

### Extraction

Once a line containing `code-from-spec: ` is found,
extract the logical name and version:

1. Take everything after `code-from-spec: ` to the end
   of the line.
2. Find the last occurrence of `@v` in that substring.
3. The logical name is everything before `@v`.
4. The version string is everything after `@v`, up to the
   next whitespace or end of line.
5. Parse the version string as an integer.

If `@v` is not found, the version is not a valid integer,
or the logical name is empty, the comment is malformed.

### Interface

```go
type SpecComment struct {
    LogicalName string
    Version     int
}

func ParseSpecComment(filePath string) (
    *SpecComment, error,
)
```

`ParseSpecComment` returns the parsed spec comment on
success. On failure, it returns an error describing what
went wrong.

### Efficiency

The parser reads line by line and retains nothing from
previous lines. It stops as soon as the pattern is found.
No intermediate state is accumulated.

### Error handling

Errors must wrap the underlying error with a descriptive
message:
- `error reading <path>: <underlying error>`
- `no spec comment found in <path>`
- `malformed spec comment in <path>: <detail>`
