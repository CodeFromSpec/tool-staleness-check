---
version: 9
parent_version: 10
implements:
  - internal/logicalnames/logicalnames_test.go
---

# TEST/tech_design/internal/logical_names

## Context

Pure function tests — no filesystem or temp directories
needed. Each test calls the function with a string input
and asserts the output.

## LogicalNameFromPath

### Spec node — root

Input: `code-from-spec/_node.md`
Expect: `"ROOT"`, `true`.

### Spec node — one level

Input: `code-from-spec/domain/_node.md`
Expect: `"ROOT/domain"`, `true`.

### Spec node — deep

Input: `code-from-spec/tech_design/logical_names/_node.md`
Expect: `"ROOT/tech_design/logical_names"`, `true`.

### Test node — root canonical

Input: `code-from-spec/default.test.md`
Expect: `"TEST"`, `true`.

### Test node — canonical

Input: `code-from-spec/domain/config/default.test.md`
Expect: `"TEST/domain/config"`, `true`.

### Test node — named

Input: `code-from-spec/domain/config/edge_cases.test.md`
Expect: `"TEST/domain/config(edge_cases)"`, `true`.

### Unrecognized path

Input: `readme.md`
Expect: `""`, `false`.

### Path without _node.md

Input: `code-from-spec/domain/config/something.md`
Expect: `""`, `false`.

### Path missing code-from-spec prefix

Input: `domain/config/_node.md`
Expect: `""`, `false`.

## PathFromLogicalName

### ROOT

Input: `"ROOT"`
Expect: `"code-from-spec/_node.md"`, `true`.

### ROOT with path

Input: `"ROOT/domain/staleness"`
Expect: `"code-from-spec/domain/staleness/_node.md"`, `true`.

### ROOT with subsection qualifier

Input: `"ROOT/domain/staleness(interface)"`
Expect: `"code-from-spec/domain/staleness/_node.md"`, `true`.

### TEST without path

Input: `"TEST"`
Expect: `"code-from-spec/default.test.md"`, `true`.

### TEST canonical

Input: `"TEST/domain/config"`
Expect: `"code-from-spec/domain/config/default.test.md"`, `true`.

### TEST named

Input: `"TEST/domain/config(edge_cases)"`
Expect: `"code-from-spec/domain/config/edge_cases.test.md"`, `true`.

### Unrecognized prefix

Input: `"UNKNOWN/something"`
Expect: `""`, `false`.

### Empty string

Input: `""`
Expect: `""`, `false`.

## LogicalNamesMatch

### Exact match

Inputs: `"ROOT/domain/config"`, `"ROOT/domain/config"`
Expect: `true`.

### Different names

Inputs: `"ROOT/domain/config"`, `"ROOT/domain/api"`
Expect: `false`.

### TEST canonical vs TEST with default

Inputs: `"TEST/domain/config"`, `"TEST/domain/config(default)"`
Expect: `true`.

### TEST with default vs TEST canonical

Inputs: `"TEST/domain/config(default)"`, `"TEST/domain/config"`
Expect: `true`.

### TEST without path vs TEST(default)

Inputs: `"TEST"`, `"TEST(default)"`
Expect: `true`.

### TEST named — same name

Inputs: `"TEST/domain/config(edge_cases)"`, `"TEST/domain/config(edge_cases)"`
Expect: `true`.

### TEST named — different name

Inputs: `"TEST/domain/config(edge_cases)"`, `"TEST/domain/config(smoke)"`
Expect: `false`.

### TEST canonical vs TEST named (non-default)

Inputs: `"TEST/domain/config"`, `"TEST/domain/config(edge_cases)"`
Expect: `false`.

### ROOT with qualifier vs ROOT without

Inputs: `"ROOT/domain/config(interface)"`, `"ROOT/domain/config"`
Expect: `true`.

### ROOT with qualifier vs ROOT without (reversed)

Inputs: `"ROOT/domain/config"`, `"ROOT/domain/config(interface)"`
Expect: `true`.

## HasParent

### ROOT

Input: `"ROOT"`
Expect: `false`, `true`.

### ROOT with path

Input: `"ROOT/domain/config"`
Expect: `true`, `true`.

### TEST without path

Input: `"TEST"`
Expect: `true`, `true`.

### TEST with path

Input: `"TEST/domain/config"`
Expect: `true`, `true`.

### TEST named

Input: `"TEST/domain/config(edge_cases)"`
Expect: `true`, `true`.

### Empty string

Input: `""`
Expect: `false`, `false`.

### Unrecognized prefix

Input: `"UNKNOWN/something"`
Expect: `false`, `false`.

## ParentLogicalName

### ROOT/x — parent is ROOT

Input: `"ROOT/domain"`
Expect: `"ROOT"`, `true`.

### ROOT/x/y — parent is ROOT/x

Input: `"ROOT/domain/config"`
Expect: `"ROOT/domain"`, `true`.

### ROOT/x/y/z — parent is ROOT/x/y

Input: `"ROOT/tech_design/logical_names"`
Expect: `"ROOT/tech_design"`, `true`.

### TEST without path — parent is ROOT

Input: `"TEST"`
Expect: `"ROOT"`, `true`.

### TEST/x — subject is ROOT/x

Input: `"TEST/domain/config"`
Expect: `"ROOT/domain/config"`, `true`.

### TEST/x(name) — subject is ROOT/x

Input: `"TEST/domain/config(edge_cases)"`
Expect: `"ROOT/domain/config"`, `true`.

### ROOT has no parent

Input: `"ROOT"`
Expect: `""`, `false`.

### Invalid input

Input: `""`
Expect: `""`, `false`.
