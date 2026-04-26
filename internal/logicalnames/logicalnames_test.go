// code-from-spec: TEST/tech_design/internal/logical_names@v8
package logicalnames

import "testing"

// ---------------------------------------------------------------------------
// LogicalNameFromPath
// ---------------------------------------------------------------------------

func TestLogicalNameFromPath_SpecNodeRoot(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/_node.md")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_SpecNodeOneLevel(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/_node.md")
	if !ok || got != "ROOT/domain" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_SpecNodeDeep(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/tech_design/logical_names/_node.md")
	if !ok || got != "ROOT/tech_design/logical_names" {
		t.Errorf("got (%q, %v), want (\"ROOT/tech_design/logical_names\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_TestNodeRootCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/default.test.md")
	if !ok || got != "TEST" {
		t.Errorf("got (%q, %v), want (\"TEST\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_TestNodeCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/default.test.md")
	if !ok || got != "TEST/domain/config" {
		t.Errorf("got (%q, %v), want (\"TEST/domain/config\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_TestNodeNamed(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/edge_cases.test.md")
	if !ok || got != "TEST/domain/config(edge_cases)" {
		t.Errorf("got (%q, %v), want (\"TEST/domain/config(edge_cases)\", true)", got, ok)
	}
}

func TestLogicalNameFromPath_UnrecognizedPath(t *testing.T) {
	got, ok := LogicalNameFromPath("readme.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestLogicalNameFromPath_PathWithoutNodeMd(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/domain/config/something.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestLogicalNameFromPath_PathMissingPrefix(t *testing.T) {
	got, ok := LogicalNameFromPath("domain/config/_node.md")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// ---------------------------------------------------------------------------
// PathFromLogicalName
// ---------------------------------------------------------------------------

func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok || got != "code-from-spec/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/_node.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/domain/staleness")
	if !ok || got != "code-from-spec/domain/staleness/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/staleness/_node.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_ROOTWithSubsectionQualifier(t *testing.T) {
	// Subsection qualifier is stripped — resolves to same file as without qualifier.
	got, ok := PathFromLogicalName("ROOT/domain/staleness(interface)")
	if !ok || got != "code-from-spec/domain/staleness/_node.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/staleness/_node.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok || got != "code-from-spec/default.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/default.test.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok || got != "code-from-spec/domain/config/default.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/config/default.test.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "code-from-spec/domain/config/edge_cases.test.md" {
		t.Errorf("got (%q, %v), want (\"code-from-spec/domain/config/edge_cases.test.md\", true)", got, ok)
	}
}

func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

// ---------------------------------------------------------------------------
// LogicalNamesMatch
// ---------------------------------------------------------------------------

func TestLogicalNamesMatch_ExactMatch(t *testing.T) {
	if !LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/config") {
		t.Error("expected true for exact match")
	}
}

func TestLogicalNamesMatch_DifferentNames(t *testing.T) {
	if LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/api") {
		t.Error("expected false for different names")
	}
}

func TestLogicalNamesMatch_TESTCanonicalVsTESTWithDefault(t *testing.T) {
	// TEST/x is an alias for TEST/x(default) — they must match.
	if !LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(default)") {
		t.Error("expected true: TEST canonical should match TEST(default)")
	}
}

func TestLogicalNamesMatch_TESTWithDefaultVsTESTCanonical(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(default)", "TEST/domain/config") {
		t.Error("expected true: TEST(default) should match TEST canonical")
	}
}

func TestLogicalNamesMatch_TESTWithoutPathVsTESTDefault(t *testing.T) {
	// TEST and TEST(default) are equivalent.
	if !LogicalNamesMatch("TEST", "TEST(default)") {
		t.Error("expected true: TEST should match TEST(default)")
	}
}

func TestLogicalNamesMatch_TESTNamedSameName(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(edge_cases)") {
		t.Error("expected true for identical named test nodes")
	}
}

func TestLogicalNamesMatch_TESTNamedDifferentName(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(smoke)") {
		t.Error("expected false for different named test nodes")
	}
}

func TestLogicalNamesMatch_TESTCanonicalVsTESTNamedNonDefault(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(edge_cases)") {
		t.Error("expected false: canonical should not match non-default named test")
	}
}

func TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithout(t *testing.T) {
	// Subsection qualifiers on ROOT names are ignored for matching.
	if !LogicalNamesMatch("ROOT/domain/config(interface)", "ROOT/domain/config") {
		t.Error("expected true: ROOT qualifier should be ignored")
	}
}

func TestLogicalNamesMatch_ROOTWithQualifierVsROOTWithoutReversed(t *testing.T) {
	if !LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/config(interface)") {
		t.Error("expected true: ROOT qualifier should be ignored (reversed)")
	}
}

// ---------------------------------------------------------------------------
// HasParent
// ---------------------------------------------------------------------------

func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if !ok || hasParent {
		t.Errorf("got (%v, %v), want (false, true)", hasParent, ok)
	}
}

func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if !ok || !hasParent {
		t.Errorf("got (%v, %v), want (true, true)", hasParent, ok)
	}
}

func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if ok || hasParent {
		t.Errorf("got (%v, %v), want (false, false)", hasParent, ok)
	}
}

func TestHasParent_UnrecognizedPrefix(t *testing.T) {
	hasParent, ok := HasParent("UNKNOWN/something")
	if ok || hasParent {
		t.Errorf("got (%v, %v), want (false, false)", hasParent, ok)
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName
// ---------------------------------------------------------------------------

func TestParentLogicalName_ROOTx_ParentIsROOT(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

func TestParentLogicalName_ROOTxy_ParentIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok || got != "ROOT/domain" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain\", true)", got, ok)
	}
}

func TestParentLogicalName_ROOTxyz_ParentIsROOTxy(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok || got != "ROOT/tech_design" {
		t.Errorf("got (%q, %v), want (\"ROOT/tech_design\", true)", got, ok)
	}
}

func TestParentLogicalName_TESTWithoutPath_ParentIsROOT(t *testing.T) {
	got, ok := ParentLogicalName("TEST")
	if !ok || got != "ROOT" {
		t.Errorf("got (%q, %v), want (\"ROOT\", true)", got, ok)
	}
}

func TestParentLogicalName_TESTx_SubjectIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain/config\", true)", got, ok)
	}
}

func TestParentLogicalName_TESTxNamed_SubjectIsROOTx(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok || got != "ROOT/domain/config" {
		t.Errorf("got (%q, %v), want (\"ROOT/domain/config\", true)", got, ok)
	}
}

func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestParentLogicalName_InvalidInput(t *testing.T) {
	got, ok := ParentLogicalName("")
	if ok || got != "" {
		t.Errorf("got (%q, %v), want (\"\", false)", got, ok)
	}
}
