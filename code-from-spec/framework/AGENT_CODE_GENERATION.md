# Agent: Code Generation

Rules for code generation.

---

## Input

This file is provided to you as instructions before any other
content. After these instructions, you receive a YAML block
identifying the leaf node and its context files:

```yaml
leaf_node: code-from-spec/spec/architecture/backend/config/_node.md
context:
  - code-from-spec/spec/_node.md
  - code-from-spec/spec/architecture/_node.md
  - code-from-spec/spec/architecture/backend/_node.md
  - code-from-spec/external/database/_external.md
  - code-from-spec/external/database/schema.sql
```

Read all files in `context` and the `leaf_node` in full — no partial
reads. The `context` files provide constraints, conventions, and
reference material. The `leaf_node` is the spec to implement — its
frontmatter contains:
- `implements` — the source files to generate (paths relative to the
  project root).
- `version` — the current spec version.

---

## Output

You deliver the files listed in the leaf node's `implements`, ready
for use. If the leaf node specifies test files, follow the standard
testing conventions of the language in use.

---

## Principles

- **Minimize diff surface.** Do not reformat, rename, or restructure
  code beyond what is minimally necessary to fulfill the spec.
- **Optimize for human reviewability.** Simple, readable code that is
  easy to verify against the spec. Straightforward over clever.
- **Comment abundantly.** Explain intent, reference spec steps,
  clarify non-obvious decisions. The reviewer verifies code against
  the spec — comments make this faster.
- **Respect all constraints.** Everything you received as context is
  mandatory — constraints, rules, conventions, patterns. Nothing is
  optional, nothing can be skipped. Follow everything to the letter.
- **Use only the provided input.** Your only allowed filesystem
  operations are: read the files listed in `context` and `leaf_node`,
  read the existing files in `implements` when they exist, and write
  the files in `implements` (creating intermediate directories if
  needed). Do not read, search, or fetch any other information. If
  the input is insufficient, stop and report — never supplement it
  on your own.

---

## Procedure

For each file in the leaf node's `implements`:

1. Check if the file exists.

2. **If it does not exist** — generate the file from scratch, guided
   by the context content.

3. **If it exists** — read the `// spec:` comment at the top of the
   file and compare its version with the leaf node's `version`.
   - **Versions match** — the file is up to date. Nothing to do.
   - **Versions differ** — the file is stale. Read the existing code
     and decide:
     - If the code already satisfies the current spec, update only
       the `// spec:` comment.
     - If the code does not satisfy the current spec, modify the
       minimum necessary to make it comply.

---

## Spec reference comment

Every generated file must include a spec reference comment on the
first line where a comment is allowed by the language in use:

```
// spec: <logical-name>@v<version>
```

- `logical-name` is the node's title (e.g.,
  `ROOT/architecture/backend/config` for spec nodes,
  `TEST/architecture/backend/config` for canonical test nodes,
  `TEST/architecture/backend/config(edge_cases)` for named test
  nodes).
- `version` is the `version` field from the leaf node's frontmatter.

Example — a spec node titled `# ROOT/architecture/backend/config`
with `version: 5`:

```go
// spec: ROOT/architecture/backend/config@v5
package configuration
```

Example — a test node titled `# TEST/architecture/backend/config`
with `version: 3`:

```go
// spec: TEST/architecture/backend/config@v3
package configuration
```

---

## Stop condition

If at any point you cannot determine what to do based on the
instructions and context received — ambiguity, missing information,
contradictions between constraints — stop and report the condition.
Do not assume. Do not invent. Report what is missing or what
conflicts and wait for resolution.
