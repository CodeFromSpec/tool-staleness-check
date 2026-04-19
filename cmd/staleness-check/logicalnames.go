// spec: ROOT/tech_design/logical_names@v3
package main

import (
	"strings"
)

// LogicalNameFromPath derives the logical name from a file path relative to
// the project root. Returns (logicalName, true) on success, or
// ("", false) if the path does not match any known pattern.
//
// Conversion rules (see ROOT/domain/specifications for full definition):
//   - code-from-spec/spec/_node.md            → ROOT
//   - code-from-spec/spec/<path>/_node.md     → ROOT/<path>
//   - code-from-spec/spec/default.test.md          → TEST
//   - code-from-spec/spec/<path>/default.test.md  → TEST/<path>
//   - code-from-spec/spec/<path>/<name>.test.md   → TEST/<path>(<name>)
//   - code-from-spec/external/<name>/_external.md → EXTERNAL/<name>
func LogicalNameFromPath(filePath string) (string, bool) {
	// Normalize backslashes to forward slashes so the function works
	// consistently on Windows and Unix.
	filePath = strings.ReplaceAll(filePath, "\\", "/")

	// --- Rule: spec node files (_node.md) ---
	if filePath == "code-from-spec/spec/_node.md" {
		// The root node has the special logical name "ROOT" with no path suffix.
		return "ROOT", true
	}
	if strings.HasPrefix(filePath, "code-from-spec/spec/") && strings.HasSuffix(filePath, "/_node.md") {
		// Strip the "code-from-spec/spec/" prefix and "/_node.md" suffix to get the path.
		path := filePath[len("code-from-spec/spec/") : len(filePath)-len("/_node.md")]
		if path == "" {
			return "", false
		}
		return "ROOT/" + path, true
	}

	// --- Rule: test node files (.test.md) ---
	if strings.HasPrefix(filePath, "code-from-spec/spec/") && strings.HasSuffix(filePath, ".test.md") {
		// Strip the "code-from-spec/spec/" prefix to isolate directory path and filename.
		rest := filePath[len("code-from-spec/spec/"):]

		// Split into directory path and filename at the last slash.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash < 0 {
			// No directory component — the test file sits directly in spec/,
			// e.g. spec/default.test.md → TEST (the root-level canonical test).
			testName := rest[:len(rest)-len(".test.md")]
			if testName == "" {
				return "", false
			}
			if testName == "default" {
				return "TEST", true
			}
			// Non-default test files directly under spec/ are not valid
			// (there is no path to attach the name qualifier to).
			return "", false
		}
		dirPath := rest[:lastSlash]
		fileName := rest[lastSlash+1:]

		// Strip the ".test.md" suffix from the filename to get the test name.
		testName := fileName[:len(fileName)-len(".test.md")]
		if testName == "" {
			return "", false
		}

		// "default" is the canonical test name — TEST/<path> without a
		// parenthesized qualifier. All other names appear as TEST/<path>(<name>).
		if testName == "default" {
			return "TEST/" + dirPath, true
		}
		return "TEST/" + dirPath + "(" + testName + ")", true
	}

	// --- Rule: external dependency files (_external.md) ---
	if strings.HasPrefix(filePath, "code-from-spec/external/") && strings.HasSuffix(filePath, "/_external.md") {
		// Strip the "code-from-spec/external/" prefix and "/_external.md" suffix
		// to get the dependency name.
		name := filePath[len("code-from-spec/external/") : len(filePath)-len("/_external.md")]
		if name == "" {
			return "", false
		}
		return "EXTERNAL/" + name, true
	}

	// No pattern matched — the path is not a recognized node file.
	return "", false
}

// PathFromLogicalName resolves a logical name to a file path relative to the
// project root. Returns (filePath, true) on success, or
// ("", false) if the logical name does not match any known pattern.
//
// Conversion rules (inverse of LogicalNameFromPath):
//   - ROOT                     → code-from-spec/spec/_node.md
//   - ROOT/<path>              → code-from-spec/spec/<path>/_node.md
//   - TEST                     → code-from-spec/spec/default.test.md
//   - TEST/<path>              → code-from-spec/spec/<path>/default.test.md
//   - TEST/<path>(<name>)      → code-from-spec/spec/<path>/<name>.test.md
//   - EXTERNAL/<name>          → code-from-spec/external/<name>/_external.md
func PathFromLogicalName(logicalName string) (string, bool) {
	// --- Rule: ROOT nodes ---
	if logicalName == "ROOT" {
		return "code-from-spec/spec/_node.md", true
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		path := logicalName[len("ROOT/"):]
		if path == "" {
			return "", false
		}
		return "code-from-spec/spec/" + path + "/_node.md", true
	}

	// --- Rule: TEST nodes ---
	if logicalName == "TEST" {
		// Bare TEST with no path → the root-level canonical test node.
		return "code-from-spec/spec/default.test.md", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}

		// Check for a parenthesized test name, e.g. TEST/x/y(edge_cases).
		// The opening paren must not be at position 0 (there must be a path
		// before it) and the closing paren must be the last character.
		parenIdx := strings.LastIndex(rest, "(")
		if parenIdx > 0 && strings.HasSuffix(rest, ")") {
			// Named test node: TEST/<path>(<name>) → code-from-spec/spec/<path>/<name>.test.md
			dirPath := rest[:parenIdx]
			testName := rest[parenIdx+1 : len(rest)-1]
			if testName == "" {
				return "", false
			}
			return "code-from-spec/spec/" + dirPath + "/" + testName + ".test.md", true
		}

		// Canonical (default) test node: TEST/<path> → code-from-spec/spec/<path>/default.test.md
		return "code-from-spec/spec/" + rest + "/default.test.md", true
	}

	// --- Rule: EXTERNAL nodes ---
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return "", false
		}
		return "code-from-spec/external/" + name + "/_external.md", true
	}

	// No prefix matched — the logical name is not recognized.
	return "", false
}

// LogicalNamesMatch compares two logical names for equivalence.
// TEST/x and TEST/x(default) are considered equal because "default" is the
// canonical test name and both forms refer to the same test node. All other
// comparisons are exact string equality.
func LogicalNamesMatch(a, b string) bool {
	// Fast path: exact string equality covers the common case.
	if a == b {
		return true
	}

	// Normalize both names so that the implicit default form TEST/<path> and
	// the explicit form TEST/<path>(default) compare as equal.
	return normalizeTestName(a) == normalizeTestName(b)
}

// normalizeTestName expands the short form TEST/<path> to the explicit form
// TEST/<path>(default) so that both representations can be compared with
// simple string equality. Bare "TEST" normalizes to "TEST(default)".
// Non-TEST names are returned unchanged.
func normalizeTestName(name string) string {
	// Bare TEST normalizes to TEST(default).
	if name == "TEST" {
		return "TEST(default)"
	}

	if !strings.HasPrefix(name, "TEST/") {
		// Not a test logical name — no normalization needed.
		return name
	}

	rest := name[len("TEST/"):]

	// If it already has a parenthesized qualifier, leave it as-is.
	if strings.Contains(rest, "(") {
		return name
	}

	// No qualifier — expand to the explicit default form.
	return name + "(default)"
}

// HasParent determines whether a logical name has a parent node. Returns
// (hasParent, ok) where ok indicates whether the input is a valid logical
// name. If ok is false, hasParent is always false.
//
// Rules (see ROOT/tech_design/logical_names for full definition):
//   - ROOT             → no parent (root of the spec tree)
//   - ROOT/<path>      → has parent
//   - TEST             → has parent (parent is ROOT)
//   - TEST/<path>      → has parent (parent is in ROOT namespace)
//   - TEST/<path>(<name>) → has parent (parent is in ROOT namespace)
//   - EXTERNAL/<name>  → no parent (external deps are standalone)
//   - anything else    → not a valid logical name (ok = false)
func HasParent(logicalName string) (hasParent, ok bool) {
	// --- ROOT namespace ---
	if logicalName == "ROOT" {
		// The root node has no parent.
		return false, true
	}
	if strings.HasPrefix(logicalName, "ROOT/") {
		path := logicalName[len("ROOT/"):]
		if path == "" {
			// "ROOT/" with nothing after it is not a valid logical name.
			return false, false
		}
		// Any ROOT node below the root has a parent.
		return true, true
	}

	// --- TEST namespace ---
	if logicalName == "TEST" {
		// Bare TEST has parent ROOT.
		return true, true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return false, false
		}
		// All TEST nodes have a parent in the ROOT namespace.
		return true, true
	}

	// --- EXTERNAL namespace ---
	if strings.HasPrefix(logicalName, "EXTERNAL/") {
		name := logicalName[len("EXTERNAL/"):]
		if name == "" {
			return false, false
		}
		// External dependencies have no parent.
		return false, true
	}

	// Bare "EXTERNAL" without a name, empty string, or any other
	// unrecognized format is not a valid logical name.
	return false, false
}

// ParentLogicalName derives the parent's logical name from a node's logical
// name. Returns (parent, true) on success, or ("", false) if the node has no
// parent. Only meaningful when HasParent returns (true, true).
//
// Rules (see ROOT/tech_design/logical_names for full definition):
//   - ROOT/<single>          → ROOT
//   - ROOT/<path>/<segment>  → ROOT/<path>   (strip last segment)
//   - TEST                   → ROOT
//   - TEST/<path>            → ROOT/<path>
//   - TEST/<path>(<name>)    → ROOT/<path>
func ParentLogicalName(logicalName string) (string, bool) {
	// --- ROOT namespace ---
	if strings.HasPrefix(logicalName, "ROOT/") {
		path := logicalName[len("ROOT/"):]
		if path == "" {
			return "", false
		}
		// Strip last segment. If no slash remains, the parent is ROOT.
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash < 0 {
			return "ROOT", true
		}
		return "ROOT/" + path[:lastSlash], true
	}

	// --- TEST namespace ---
	if logicalName == "TEST" {
		// Bare TEST's parent is ROOT.
		return "ROOT", true
	}
	if strings.HasPrefix(logicalName, "TEST/") {
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		// Strip any parenthesized test name qualifier first.
		parenIdx := strings.LastIndex(rest, "(")
		if parenIdx > 0 && strings.HasSuffix(rest, ")") {
			rest = rest[:parenIdx]
		}
		// The parent of TEST/<path> is ROOT/<path>.
		return "ROOT/" + rest, true
	}

	// ROOT (no parent), EXTERNAL (no parent), or unrecognized.
	return "", false
}
