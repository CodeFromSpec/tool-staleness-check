// code-from-spec: TEST/tech_design/internal/logical_names@v10
package logicalnames

import "testing"

// ---------------------------------------------------------------------------
// LogicalNameFromPath
// ---------------------------------------------------------------------------

// TestLogicalNameFromPath_SpecNodeRoot verifies that the root _node.md
// resolves to the "ROOT" logical name.
func TestLogicalNameFromPath_SpecNodeRoot(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/_node.md")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_SpecNodeOneLevel verifies a single-segment
// spec node path resolves to ROOT/<segment>.
func TestLogicalNameFromPath_SpecNodeOneLevel(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/_node.md")
	if !ok || got != "ROOT/domain" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_SpecNodeDeep verifies a multi-segment spec
// node path resolves to ROOT/<path>.
func TestLogicalNameFromPath_SpecNodeDeep(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/tech_design/logical_names/_node.md")
	if !ok || got != "ROOT/tech_design/logical_names" {
		t.Errorf("got (%q, %v), want (\"ROOT/tech_design/logical_names\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_TestNodeRootCanonical verifies that the root
// default.test.md resolves to "TEST".
func TestLogicalNameFromPath_TestNodeRootCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/default.test.md")
	if !ok || got != "TEST" {
		t.Errorf("got (%q, %v), want (\"TEST\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_TestNodeCanonical verifies that a default.test.md
// inside a path resolves to TEST/<path>.
func TestLogicalNameFromPath_TestNodeCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/default.test.md")
	if !ok || got != "TEST/domain/config" {
		t.Errorf("got (%q, %v), want (\"TEST/domain/config\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_TestNodeNamed verifies that a named .test.md
// resolves to TEST/<path>(<name>).
func TestLogicalNameFromPath_TestNodeNamed(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/edge_cases.test.md")
	if !ok || got != "TEST/domain/config(edge_cases)" {
		t.Errorf("got (%q, %v), want (\"TEST/domain/config(edge_cases)\", true)", got, ok)
	}
}

// TestLogicalNameFromPath_UnrecognizedPath verifies that a path that does
// not match any known pattern returns ("", false).
func TestLogicalNameFromPath_UnrecognizedPath(t *testing.T) {
	got, ok := LogicalNameFromPath("readme.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// TestLogicalNameFromPath_PathWithoutNodeMd verifies that a .md file
// inside code-from-spec that is neither _node.md nor *.test.md returns
// ("", false).
func TestLogicalNameFromPath_PathWithoutNodeMd(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/something.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// TestLogicalNameFromPath_PathMissingPrefix verifies that a path that
// looks like a spec node path but lacks the code-from-spec/ prefix
// returns ("", false).
func TestLogicalNameFromPath_PathMissingPrefix(t *testing.T) {
	got, ok := LogicalNameFromPath("domain/config/_node.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// ---------------------------------------------------------------------------
// PathFromLogicalName
// ---------------------------------------------------------------------------

// TestPathFromLogicalName_ROOT verifies that "ROOT" maps to the root
// _node.md path.
func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok || got != "code-from-spec/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/_node.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_ROOTWithPath verifies that "ROOT/<path>"
// maps to code-from-spec/<path>/_node.md.
func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/domain/staleness")
	if !ok || got != "code-from-spec/domain/staleness/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/staleness/_node.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_ROOTWithSubsectionQualifier verifies that a
// subsection qualifier on a ROOT name is stripped and the same file
// path is produced as without the qualifier.
func TestPathFromLogicalName_ROOTWithSubsectionQualifier(t *testing.T) {
	// Subsection qualifier is stripped — resolves to same file as without qualifier.
	got, ok := PathFromLogicalName("ROOT/domain/staleness(interface)")
	if !ok || got != "code-from-spec/domain/staleness/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/staleness/_node.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_TESTWithoutPath verifies that "TEST" maps to
// the root default.test.md.
func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok || got != "code-from-spec/default.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/default.test.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_TESTCanonical verifies that "TEST/<path>"
// maps to code-from-spec/<path>/default.test.md.
func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok || got != "code-from-spec/domain/config/default.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/config/default.test.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_TESTNamed verifies that "TEST/<path>(<name>)"
// maps to code-from-spec/<path>/<name>.test.md.
func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "code-from-spec/domain/config/edge_cases.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/config/edge_cases.test.md\", true)", got, ok)
	}
}

// TestPathFromLogicalName_UnrecognizedPrefix verifies that a logical
// name with an unknown prefix returns ("", false).
func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// TestPathFromLogicalName_EmptyString verifies that an empty string
// returns ("", false).
func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// ---------------------------------------------------------------------------
// LogicalNamesMatch
// ---------------------------------------------------------------------------

// TestLogicalNamesMatch_ExactMatch verifies that two identical ROOT
// names match.
func TestLogicalNamesMatch_ExactMatch(t *testing.T) {
	if !LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/config") {
		t.Error("expected true for exact match")
	}
}

// TestLogicalNamesMatch_DifferentNames verifies that two distinct ROOT
// names do not match.
func TestLogicalNamesMatch_DifferentNames(t *testing.T) {
	if LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/api") {
		t.Error("expected false for different names")
	}
}

// TestLogicalNamesMatch_TESTCanonicalVsTESTWithDefault verifies that
// the canonical form TEST/x matches the aliased form TEST/x(default).
func TestLogicalNamesMatch_TESTCanonicalVsTESTWithDefault(t *testing.T) {
	// TEST/x is an alias for TEST/x(default) — they must match.
	if !LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(default)") {
		t.Error("expected true: TEST canonical should match TEST(default)")
	}
}

// TestLogicalNamesMatch_TESTWithDefaultVsTESTCanonical verifies the
// reverse direction of the canonical/default alias.
func TestLogicalNamesMatch_TESTWithDefaultVsTESTCanonical(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(default)", "TEST/domain/config") {
		t.Error("expected true: TEST(default) should match TEST canonical")
	}
}

// TestLogicalNamesMatch_TESTWithoutPathVsTESTDefault verifies that
// the root TEST name matches TEST(default).
func TestLogicalNamesMatch_TESTWithoutPathVsTESTDefault(t *testing.T) {
	// TEST and TEST(default) are equivalent.
	if !LogicalNamesMatch("TEST", "TEST(default)") {
		t.Error("expected true: TEST should match TEST(default)")
	}
}

// TestLogicalNamesMatch_TESTNamedSameName verifies that two identical
// named test nodes match.
func TestLogicalNamesMatch_TESTNamedSameName(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(edge_cases)") {
		t.Error("expected true for identical named test nodes")
	}
}

// TestLogicalNamesMatch_TESTNamedDifferentName verifies that two named
// test nodes with different names do not match.
func TestLogicalNamesMatch_TESTNamedDifferentName(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(smoke)") {
		t.Error("expected false for different named test nodes")
	}
}

// TestLogicalNamesMatch_TESTCanonicalVsTESTNamedNonDefault verifies
// that the canonical form does not match a non-default named test node.
func TestLogicalNamesMatch_TESTCanonicalVsTESTNamedNonDefault(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(edge_cases)") {
		t.Error("expected false: canonical should not match non-default named test")
	}
}

// TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithout verifies that
// a ROOT name with a subsection qualifier matches the same name without.
func TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithout(t *testing.T) {
	// Subsection qualifiers on ROOT names are ignored for matching.
	if !LogicalNamesMatch("ROOT/domain/config(interface)", "ROOT/domain/config") {
		t.Error("expected true: ROOT qualifier should be ignored")
	}
}

// TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithoutReversed verifies
// the reverse direction of the ROOT qualifier-ignoring rule.
func TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithoutReversed(t *testing.T) {
	if !LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/config(interface)") {
		t.Error("expected true: ROOT qualifier should be ignored (reversed)")
	}
}

// ---------------------------------------------------------------------------
// HasParent
// ---------------------------------------------------------------------------

// TestHasParent_ROOT verifies that ROOT has no parent but is a valid
// logical name.
func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if !ok || hasParent {
		t.Errorf("got (%v, %v), want (false, true)", hasParent, ok)
	}
}

// TestHasParent_ROOTWithPath verifies that ROOT/<path> has a parent.
func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

// TestHasParent_TESTWithoutPath verifies that TEST (root test node)
// has a parent (ROOT).
func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

// TestHasParent_TESTWithPath verifies that TEST/<path> has a parent.
func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

// TestHasParent_TESTNamed verifies that TEST/<path>(<name>) has a
// parent.
func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

// TestHasParent_EmptyString verifies that an empty string is not a
// valid logical name.
func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if ok || hasParent {
		t.Errorf("got (%v, %v), want (false, false)", hasParent, ok)
	}
}

// TestHasParent_UnrecognizedPrefix verifies that a logical name with
// an unknown prefix is not valid.
func TestHasParent_UnrecognizedPrefix(t *testing.T) {
	hasParent, ok := HasParent("UNKNOWN/something")
	if ok || hasParent {
		t.Errorf("got (%v, %v), want (false, false)", hasParent, ok)
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName
// ---------------------------------------------------------------------------

// TestParentLogicalName_ROOTx_ParentIsROOT verifies that ROOT/<x>
// has ROOT as its parent.
func TestParentLogicalName_ROOTx_ParentIsROOT(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

// TestParentLogicalName_ROOTxy_ParentIsROOTx verifies that ROOT/<x>/<y>
// has ROOT/<x> as its parent.
func TestParentLogicalName_ROOTxy_ParentIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok || got != "ROOT/domain" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain\", true)", got, ok)
	}
}

// TestParentLogicalName_ROOTxyz_ParentIsROOTxy verifies that a three-
// segment ROOT path has a two-segment ROOT path as its parent.
func TestParentLogicalName_ROOTxyz_ParentIsROOTxy(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok || got != "ROOT/tech_design" {
		t.Errorf("got (%q, %v), want (\"ROOT/tech_design\", true)", got, ok)
	}
}

// TestParentLogicalName_TESTWithoutPath_ParentIsROOT verifies that
// the root test node TEST has ROOT as its parent/subject.
func TestParentLogicalName_TESTWithoutPath_ParentIsROOT(t *testing.T) {
	got, ok := ParentLogicalName("TEST")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

// TestParentLogicalName_TESTx_SubjectIsROOTx verifies that TEST/<path>
// returns ROOT/<path> as its subject (parent in ROOT namespace).
func TestParentLogicalName_TESTx_SubjectIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain/config\", true)", got, ok)
	}
}

// TestParentLogicalName_TESTxNamed_SubjectIsROOTx verifies that
// TEST/<path>(<name>) returns ROOT/<path> as its subject, stripping
// the test name qualifier.
func TestParentLogicalName_TESTxNamed_SubjectIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain/config\", true)", got, ok)
	}
}

// TestParentLogicalName_ROOTHasNoParent verifies that ROOT itself has
// no parent and returns ("", false).
func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// TestParentLogicalName_InvalidInput verifies that an empty string
// input returns ("", false).
func TestParentLogicalName_InvalidInput(t *testing.T) {
	got, ok := ParentLogicalName("")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}
