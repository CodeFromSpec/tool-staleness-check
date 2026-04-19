---
version: 6
---

# Code From Spec

**Code from Spec** is a methodology where code is a generated
artifact, not the source of truth. The source of truth is a hierarchy
of specification files. To change behavior, you change the spec and
regenerate. You never edit generated code directly.

This methodology is designed for AI agent participation at every
stage — writing specs, managing versions, detecting staleness,
running resyncs, generating code, and assisting non-technical
contributors with spec authoring.

---

## The Model

Specifications are organized as a tree. Each node adds precision
to its parent — business intent at the root, implementation
contracts at the leaves. Only leaf nodes generate code.

```
root/
└── payments/
    └── fees/
        ├── calculation/   ← leaf, implemented
        └── rounding/      ← leaf, implemented
```

---

## Structure

```
/
  code-from-spec/
    CODE_FROM_SPEC.md      ← this file
    spec/                  ← spec tree
    external/              ← external dependencies
    framework/             ← framework documentation
```

See `framework/SPECIFICATIONS.md` for the rules on writing
specifications and `framework/EXTERNAL_DEPENDENCIES.md` for the
rules on external dependencies.

---

## Orchestration

The main agent (orchestrator) does not verify staleness or generate
code directly. It dispatches subagents, each with a specific role
and a self-contained set of instructions.

Each subagent receives:
- An instruction file (`AGENT_*.md`) defining its rules, procedure,
  and allowed operations.
- A structured input (YAML) with the specific data to process.

Subagents operate only with what they receive. They do not explore
the filesystem, search for files, or read anything beyond what
their instructions allow. The orchestrator is responsible for
assembling the correct input — if the input is wrong or incomplete,
the subagent's output will be wrong.

---

## Code Generation

The orchestrator assembles the context for each code generation
agent by building the **chain** — the ordered sequence of ancestor
`_node.md` files from root to the target leaf node, followed by
any `depends_on` content.

For spec `depends_on`: the referenced node file. For external
`depends_on`: the `_external.md` plus all files in the dependency
folder (filtered by `filter` if specified).

Example — implementing `ROOT/payments/fees/calculation`:

```
ROOT                          (spec/_node.md)
ROOT/payments                 (spec/payments/_node.md)
ROOT/payments/fees            (spec/payments/fees/_node.md)
ROOT/payments/fees/calculation (spec/payments/fees/calculation/_node.md)
EXTERNAL/database             (_external.md + schema.sql)
```

The chain is the complete context. Nothing outside the chain is
needed. Nothing inside the chain is redundant.

See `framework/AGENT_CODE_GENERATION.md` for the agent's rules and
procedures.

---

## Staleness Verification

See `framework/VERSIONING_AND_STALENESS.md` for versioning rules and
staleness conditions.

Spec staleness and code staleness are verified separately. Spec
staleness detects nodes whose dependencies changed — it must be
resolved first because code is generated from specs. Verifying
code against stale specs produces meaningless results.

See `framework/AGENT_SPEC_STALENESS.md` and `framework/AGENT_CODE_STALENESS.md` for
the verification procedures.

---

## Resync

When something changes — a spec is updated, an external dependency
changes, or a full regeneration is needed — run a resync:

1. **Detect and resolve spec staleness** — run spec staleness
   verification (see `framework/AGENT_SPEC_STALENESS.md`). For each stale
   node, revise the spec content if needed and update the
   declared versions. If changes introduce ambiguity or require
   human judgment, stop and consult the user. Process in
   dependency order: parents before children, dependencies before
   dependents.

2. **Generate code** — run code staleness verification (see
   `framework/AGENT_CODE_STALENESS.md`). For each stale file, dispatch a
   code generation agent (see `framework/AGENT_CODE_GENERATION.md`).

3. **Verify** — build and run tests. If anything fails, trace
   back to the spec and correct it. Do not patch the generated
   code.

---

## Techniques

See `framework/TECHNIQUES.md` for practices and patterns for working
effectively with the spec tree.

---

## Formatting

All files in `code-from-spec/` must wrap lines at **80 columns
maximum** for readability
in terminals, diffs, and non-rendered views. Use as many columns as
possible without breaking individual words — fill lines close to 80,
do not wrap conservatively at 60. Table rows and code blocks are
exempt — they may exceed 80 columns.
