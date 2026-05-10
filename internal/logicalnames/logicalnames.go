// code-from-spec: ROOT/tech_design/internal/logical_names@v11
// Package logicalnames centralizes conversion between logical names and file
// paths, and logical name comparison. Used by the discovery, spec staleness,
// and code staleness modules.
//
// Logical name namespaces:
//   - ROOT/* — spec nodes (e.g., ROOT, ROOT/x, ROOT/x/y)
//   - TEST/* — test nodes (e.g., TEST, TEST/x, TEST/x(name))
//
// File-path roots:
//   - Spec nodes live under code-from-spec/ as _node.md files.
//   - Test nodes live under code-from-spec/ as <name>.test.md files.
package logicalnames

import (
	"path"
	"strings"
)

// specDir is the root directory where all spec files live, relative to the
// project root.
const specDir = "code-from-spec"

// nodeFile is the filename used for spec nodes inside their directory.
const nodeFile = "_node.md"

// testSuffix is the file extension shared by all test node files.
const testSuffix = ".test.md"

// LogicalNameFromPath derives the logical name from a file path that is
// relative to the project root. Returns ("", false) if the path does not
// match any known pattern.
//
// Mapping table (examples):
//
//	code-from-spec/_node.md                 → ROOT
//	code-from-spec/x/_node.md               → ROOT/x
//	code-from-spec/x/y/_node.md             → ROOT/x/y
//	code-from-spec/default.test.md          → TEST
//	code-from-spec/x/default.test.md        → TEST/x
//	code-from-spec/x/name.test.md           → TEST/x(name)
func LogicalNameFromPath(filePath string) (string, bool) {
	// Normalize path separators to forward slashes so that Windows paths
	// (using backslashes) are handled consistently.
	normalized := normalizeSeparators(filePath)

	// The path must start with the spec directory followed by a separator.
	prefix := specDir + "/"
	if !strings.HasPrefix(normalized, prefix) {
		return "", false
	}

	// Strip the "code-from-spec/" prefix to get the remainder.
	rest := normalized[len(prefix):]

	// --- Spec nodes (_node.md) ---

	// Root spec node: code-from-spec/_node.md
	if rest == nodeFile {
		return "ROOT", true
	}

	// Non-root spec node: code-from-spec/<path>/_node.md
	nodeFileSuffix := "/" + nodeFile
	if strings.HasSuffix(rest, nodeFileSuffix) {
		// The path segment between the spec dir and /_node.md is the logical path.
		p := rest[:len(rest)-len(nodeFileSuffix)]
		if p == "" {
			return "", false
		}
		return "ROOT/" + p, true
	}

	// --- Test nodes (<name>.test.md) ---

	if strings.HasSuffix(rest, testSuffix) {
		// Split into directory and filename using path.Dir / path.Base so that
		// we handle single-level and multi-level paths uniformly.
		dir := path.Dir(rest)  // "." if file is directly under specDir
		base := path.Base(rest)
		// Remove the .test.md suffix to obtain the test name.
		name := base[:len(base)-len(testSuffix)]

		if dir == "." {
			// File is directly under code-from-spec/ — no subdirectory path.
			// Only "default" maps to a valid TEST logical name at this level.
			// The spec shows TEST (no path) only for code-from-spec/default.test.md.
			// There is no TEST(<name>) form without a path segment.
			if name == "default" {
				return "TEST", true
			}
			return "", false
		}

		// File is in a subdirectory — dir holds the relative path (e.g., "x/y").
		if name == "default" {
			// code-from-spec/<path>/default.test.md → TEST/<path>
			return "TEST/" + dir, true
		}
		// code-from-spec/<path>/<name>.test.md → TEST/<path>(<name>)
		return "TEST/" + dir + "(" + name + ")", true
	}

	// No known pattern matched.
	return "", false
}

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returns ("", false) if the input does not match any known
// pattern.
//
// A subsection qualifier on a ROOT name (e.g., ROOT/x/y(z)) is stripped
// before resolution — ROOT/x/y(z) resolves to the same file as ROOT/x/y.
// For TEST names, a parenthesized qualifier identifies the test file name:
// TEST/x(name) → code-from-spec/x/name.test.md.
//
// Mapping table (examples):
//
//	ROOT              → code-from-spec/_node.md
//	ROOT/x/y          → code-from-spec/x/y/_node.md
//	ROOT/x/y(z)       → code-from-spec/x/y/_node.md  (qualifier stripped)
//	TEST              → code-from-spec/default.test.md
//	TEST/x            → code-from-spec/x/default.test.md
//	TEST/x(name)      → code-from-spec/x/name.test.md
func PathFromLogicalName(logicalName string) (string, bool) {
	if logicalName == "" {
		return "", false
	}

	switch {
	// ---- ROOT namespace ----

	case logicalName == "ROOT":
		// The root spec node.
		return specDir + "/" + nodeFile, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			// "ROOT/" with nothing after — invalid.
			return "", false
		}
		// Strip any subsection qualifier before resolving.
		// e.g., "x/y(interface)" → "x/y"
		rest = stripQualifier(rest)
		if rest == "" {
			return "", false
		}
		return specDir + "/" + rest + "/" + nodeFile, true

	// ---- TEST namespace ----

	case logicalName == "TEST":
		// The canonical test node at the root level.
		return specDir + "/default" + testSuffix, true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			// "TEST/" with nothing after — invalid.
			return "", false
		}
		// Separate the path from the optional test-name qualifier.
		// e.g., "x/y(edge_cases)" → p="x/y", name="edge_cases"
		//        "x/y"            → p="x/y", name=""
		p, name := parseTestQualifier(rest)
		if name == "" {
			// No qualifier — refers to the default test file.
			return specDir + "/" + p + "/default" + testSuffix, true
		}
		// Named test file.
		return specDir + "/" + p + "/" + name + testSuffix, true

	default:
		return "", false
	}
}

// LogicalNamesMatch compares two logical names for equivalence.
//
// Two special equivalence rules apply:
//  1. TEST/x and TEST/x(default) are the same — the bare form is an alias
//     for the explicit (default) form. Named variants like TEST/x(edge_cases)
//     only match themselves.
//  2. ROOT/x(qualifier) and ROOT/x are the same — subsection qualifiers on
//     ROOT names are ignored for matching purposes.
//
// All other comparisons are exact string equality.
func LogicalNamesMatch(a, b string) bool {
	return normalizeLogicalName(a) == normalizeLogicalName(b)
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input is a valid
// logical name at all.
//
// Rules:
//   - ROOT           → (false, true)   — the root has no parent
//   - ROOT/<path>    → (true,  true)   — all other spec nodes have a parent
//   - TEST           → (true,  true)   — parent is ROOT
//   - TEST/<path>    → (true,  true)   — parent is the corresponding ROOT node
//   - TEST/<path>(n) → (true,  true)   — same
//   - ""             → (false, false)  — not a valid logical name
//   - anything else  → (false, false)  — not a valid logical name
func HasParent(logicalName string) (hasParent, ok bool) {
	switch {
	case logicalName == "ROOT":
		// The root spec node has no parent by definition.
		return false, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			// "ROOT/" alone is invalid.
			return false, false
		}
		return true, true

	case logicalName == "TEST":
		// TEST's parent/subject is ROOT.
		return true, true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			// "TEST/" alone is invalid.
			return false, false
		}
		return true, true

	default:
		// Empty string or any unrecognized form.
		return false, false
	}
}

// ParentLogicalName derives the parent's logical name from a node's logical
// name. For test nodes, this returns the subject's logical name (the ROOT
// node being tested). Returns ("", false) if the node has no parent.
//
// Rules:
//   - ROOT           → ("", false)       — no parent
//   - ROOT/x         → ("ROOT", true)
//   - ROOT/x/y       → ("ROOT/x", true)
//   - TEST           → ("ROOT", true)
//   - TEST/x         → ("ROOT/x", true)
//   - TEST/x(name)   → ("ROOT/x", true)
func ParentLogicalName(logicalName string) (string, bool) {
	switch {
	case logicalName == "ROOT":
		// Root has no parent.
		return "", false

	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		// Strip any subsection qualifier before computing the parent.
		// e.g., "x/y(z)" → "x/y"
		rest = stripQualifier(rest)

		// Find the last slash to identify the boundary between parent path and
		// this node's segment.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash == -1 {
			// Single segment (e.g., ROOT/x) — parent is ROOT.
			return "ROOT", true
		}
		// Multiple segments (e.g., ROOT/x/y) — parent is ROOT/<prefix>.
		return "ROOT/" + rest[:lastSlash], true

	case logicalName == "TEST":
		// The subject of the root-level test node is ROOT.
		return "ROOT", true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		// For test nodes, strip the test-name qualifier to get the path, then
		// map that path back into the ROOT namespace.
		// e.g., "x/y(name)" → path="x/y" → "ROOT/x/y"
		//        "x/y"       → path="x/y" → "ROOT/x/y"
		p, _ := parseTestQualifier(rest)
		return "ROOT/" + p, true

	default:
		return "", false
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// normalizeSeparators converts backslashes to forward slashes so that paths
// from Windows file systems are handled uniformly.
func normalizeSeparators(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// stripQualifier removes a trailing parenthesized qualifier from a path
// segment. For example, "x/y(interface)" becomes "x/y". If the string does
// not end with a closing parenthesis or has no opening parenthesis, it is
// returned unchanged.
//
// Note: We use LastIndex so that only the final parenthesized group is
// stripped, leaving any parentheses embedded in path segments intact
// (although the spec does not currently use such forms).
func stripQualifier(s string) string {
	// Must end with ")" to be a valid qualifier.
	if !strings.HasSuffix(s, ")") {
		return s
	}
	idx := strings.LastIndex(s, "(")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// parseTestQualifier splits a TEST path remainder into (path, name).
// The name is the content inside the final parentheses, e.g.:
//
//	"x/y(edge_cases)" → ("x/y", "edge_cases")
//	"x/y"             → ("x/y", "")
//
// If no qualifier is present, name is returned as an empty string.
func parseTestQualifier(s string) (p, name string) {
	// Must end with ")" to have a qualifier.
	if !strings.HasSuffix(s, ")") {
		return s, ""
	}
	idx := strings.LastIndex(s, "(")
	if idx == -1 {
		return s, ""
	}
	p = s[:idx]
	name = s[idx+1 : len(s)-1]
	return p, name
}

// normalizeLogicalName produces a canonical form of a logical name used
// internally by LogicalNamesMatch:
//
//   - ROOT names: subsection qualifiers are stripped.
//   - TEST names: bare TEST/<path> is expanded to TEST/<path>(default),
//     and the bare TEST is expanded to TEST(default).
//   - All other names are returned unchanged.
//
// This ensures that the two equivalence rules in LogicalNamesMatch are
// implemented through simple string equality on normalized forms.
func normalizeLogicalName(name string) string {
	switch {
	case name == "ROOT":
		return name

	case strings.HasPrefix(name, "ROOT/"):
		rest := name[len("ROOT/"):]
		// Strip any subsection qualifier — ROOT/x(z) ≡ ROOT/x.
		return "ROOT/" + stripQualifier(rest)

	case name == "TEST":
		// TEST is an alias for the default test at the root level.
		// Normalize to a sentinel so it only matches itself (and its explicit
		// default alias, should one ever be constructed).
		return "TEST(default)"

	case strings.HasPrefix(name, "TEST/"):
		rest := name[len("TEST/"):]
		p, testName := parseTestQualifier(rest)
		if testName == "" {
			// Bare form — expand to explicit default.
			testName = "default"
		}
		return "TEST/" + p + "(" + testName + ")"

	default:
		return name
	}
}
