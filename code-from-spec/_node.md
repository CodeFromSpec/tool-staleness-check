---
version: 9
---

# ROOT

## Intent

Command-line tool that performs staleness verification for
a Code from Spec project.

## Context

The Code from Spec methodology originally delegated
staleness verification to AI agents. In practice,
agent-based verification is expensive in tokens and
unreliable — LLMs can misread frontmatter, skip nodes, or
hallucinate statuses. The verification logic is fully
deterministic: read frontmatter, compare version numbers,
report results. A compiled tool eliminates the cost and
reliability problems while producing identical results.

## Contracts

The tool is invoked from the project root:

```
staleness-check
```

Without arguments, the tool runs staleness verification
and emits YAML results to stdout.

With any argument (e.g., `--help`, `-h`, or anything
else), the tool prints a help message describing what it
does and the format of its output, then exits with
code 0.

## Constraints

- The tool must not modify any file.
