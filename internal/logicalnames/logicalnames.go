// code-from-spec: ROOT/tech_design/internal/logical_names@v9
package logicalnames

import (
	"path"
	"strings"
)

// specDir is the root directory where all spec files live.
const specDir = "code-from-spec"

// nodeFile is the filename for spec nodes.
const nodeFile = "_node.md"

// testSuffix is the extension for test node files.
const testSuffix = ".test.md"

// LogicalNameFromPath derives the logical name from a file path
// relative to the project root. Returns ("", false) if the path
// does not match any known pattern.
//
// Rules:
//   - code-from-spec/_node.md                    → ROOT
//   - code-from-spec/<path>/_node.md              → ROOT/<path>
//   - code-from-spec/default.test.md              → TEST
//   - code-from-spec/<path>/default.test.md       → TEST/<path>
//   - code-from-spec/<path>/<name>.test.md        → TEST/<path>(<name>)
func LogicalNameFromPath(filePath string) (string, bool) {
	// Normalize to forward slashes for consistent handling.
	normalized := filepath(filePath)

	// The path must start with the spec directory.
	if !strings.HasPrefix(normalized, specDir+"/") {
		return "", false
	}

	// Strip the spec directory prefix.
	rest := normalized[len(specDir)+1:]

	// Check if this is a _node.md file (spec node).
	if rest == nodeFile {
		// Root node.
		return "ROOT", true
	}
	if strings.HasSuffix(rest, "/"+nodeFile) {
		// Non-root spec node: strip the trailing /_node.md to get the path.
		p := rest[:len(rest)-len("/"+nodeFile)]
		return "ROOT/" + p, true
	}

	// Check if this is a test node (.test.md file).
	if strings.HasSuffix(rest, testSuffix) {
		// Get directory and filename.
		dir := path.Dir(rest)
		base := path.Base(rest)
		name := base[:len(base)-len(testSuffix)]

		if dir == "." {
			// File is directly in code-from-spec/ (no subdirectory).
			if name == "default" {
				return "TEST", true
			}
			// A non-default test at the root level is not a valid pattern
			// because TEST(<name>) with no path doesn't match the spec rules.
			// The spec only shows TEST/<path>(<name>) — there's no TEST(<name>).
			return "", false
		}

		// File is in a subdirectory.
		if name == "default" {
			return "TEST/" + dir, true
		}
		return "TEST/" + dir + "(" + name + ")", true
	}

	return "", false
}

// PathFromLogicalName resolves a logical name to a file path
// relative to the project root. Returns ("", false) if the input
// does not match any known pattern.
//
// A subsection qualifier (e.g., ROOT/x/y(z)) is stripped before
// resolution for ROOT names. For TEST names, the parenthesized
// part is the test name.
func PathFromLogicalName(logicalName string) (string, bool) {
	if logicalName == "" {
		return "", false
	}

	switch {
	case logicalName == "ROOT":
		return specDir + "/" + nodeFile, true

	case strings.HasPrefix(logicalName, "ROOT/"):
		// Strip the ROOT/ prefix.
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return "", false
		}
		// Strip subsection qualifier if present: ROOT/x/y(z) → ROOT/x/y
		rest = stripQualifier(rest)
		if rest == "" {
			return "", false
		}
		return specDir + "/" + rest + "/" + nodeFile, true

	case logicalName == "TEST":
		return specDir + "/default" + testSuffix, true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		// Check for a test name qualifier: TEST/<path>(<name>)
		p, name := parseTestQualifier(rest)
		if name == "" {
			// No qualifier — this is the default test.
			return specDir + "/" + p + "/default" + testSuffix, true
		}
		return specDir + "/" + p + "/" + name + testSuffix, true

	default:
		return "", false
	}
}

// LogicalNamesMatch compares two logical names for equivalence.
//
// Special rules:
//   - TEST/x and TEST/x(default) are equivalent.
//   - ROOT/x(qualifier) and ROOT/x are equivalent (subsection
//     qualifiers on ROOT names are ignored).
func LogicalNamesMatch(a, b string) bool {
	return normalizeLogicalName(a) == normalizeLogicalName(b)
}

// HasParent determines whether a logical name has a parent node.
// Returns (hasParent, ok) where ok indicates whether the input
// is a valid logical name.
//
//   - ROOT           → (false, true)
//   - ROOT/<path>    → (true, true)
//   - TEST (and all TEST variants) → (true, true)
//   - anything else  → (false, false)
func HasParent(logicalName string) (hasParent, ok bool) {
	switch {
	case logicalName == "ROOT":
		return false, true
	case strings.HasPrefix(logicalName, "ROOT/"):
		rest := logicalName[len("ROOT/"):]
		if rest == "" {
			return false, false
		}
		return true, true
	case logicalName == "TEST":
		return true, true
	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return false, false
		}
		return true, true
	default:
		return false, false
	}
}

// ParentLogicalName derives the parent's logical name from a
// node's logical name. For test nodes, returns the subject's
// logical name. Returns ("", false) if the node has no parent.
//
//   - ROOT/x         → ROOT
//   - ROOT/x/y       → ROOT/x
//   - TEST            → ROOT
//   - TEST/x          → ROOT/x
//   - TEST/x(name)    → ROOT/x
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
		// Strip any subsection qualifier first.
		rest = stripQualifier(rest)
		// Find the last slash to determine the parent path.
		lastSlash := strings.LastIndex(rest, "/")
		if lastSlash == -1 {
			// Only one segment — parent is ROOT.
			return "ROOT", true
		}
		return "ROOT/" + rest[:lastSlash], true

	case logicalName == "TEST":
		// Subject of TEST is ROOT.
		return "ROOT", true

	case strings.HasPrefix(logicalName, "TEST/"):
		rest := logicalName[len("TEST/"):]
		if rest == "" {
			return "", false
		}
		// Strip test name qualifier: TEST/<path>(<name>) → <path>
		p, _ := parseTestQualifier(rest)
		return "ROOT/" + p, true

	default:
		return "", false
	}
}

// --- Internal helpers ---

// filepath normalizes a file path to use forward slashes.
func filepath(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// stripQualifier removes a trailing parenthesized qualifier from
// a path segment. E.g., "x/y(z)" → "x/y".
// If no qualifier is present, returns the input unchanged.
func stripQualifier(s string) string {
	idx := strings.LastIndex(s, "(")
	if idx == -1 {
		return s
	}
	if !strings.HasSuffix(s, ")") {
		return s
	}
	return s[:idx]
}

// parseTestQualifier splits "path(name)" into ("path", "name").
// If there is no qualifier, returns (input, "").
func parseTestQualifier(s string) (string, string) {
	idx := strings.LastIndex(s, "(")
	if idx == -1 {
		return s, ""
	}
	if !strings.HasSuffix(s, ")") {
		return s, ""
	}
	p := s[:idx]
	name := s[idx+1 : len(s)-1]
	return p, name
}

// normalizeLogicalName produces a canonical form for comparison.
//   - ROOT names: strip subsection qualifiers.
//   - TEST names: expand bare TEST/<path> to TEST/<path>(default).
func normalizeLogicalName(name string) string {
	switch {
	case name == "ROOT":
		return name
	case strings.HasPrefix(name, "ROOT/"):
		rest := name[len("ROOT/"):]
		return "ROOT/" + stripQualifier(rest)
	case name == "TEST":
		// TEST is an alias for TEST with default — but there's no path,
		// so we just keep it as "TEST(default)" for normalization.
		return "TEST(default)"
	case strings.HasPrefix(name, "TEST/"):
		rest := name[len("TEST/"):]
		p, testName := parseTestQualifier(rest)
		if testName == "" {
			testName = "default"
		}
		return "TEST/" + p + "(" + testName + ")"
	default:
		return name
	}
}
