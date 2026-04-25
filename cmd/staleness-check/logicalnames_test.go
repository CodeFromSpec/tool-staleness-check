// spec: TEST/tech_design/logical_names@v3
package main

import (
	"testing"
)

// ---------------------------------------------------------------------------
// LogicalNameFromPath tests
// ---------------------------------------------------------------------------

// TestLogicalNameFromPath_SpecNodeRoot verifies that the root spec node
// (code-from-spec/spec/_node.md) maps to the logical name "ROOT".
func TestLogicalNameFromPath_SpecNodeRoot(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/_node.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/_node.md")
	}
	if got != "ROOT" {
		t.Errorf("expected %q, got %q", "ROOT", got)
	}
}

// TestLogicalNameFromPath_SpecNodeOneLevel verifies a single-level spec node
// path (code-from-spec/spec/domain/_node.md) maps to ROOT/domain.
func TestLogicalNameFromPath_SpecNodeOneLevel(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/domain/_node.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/domain/_node.md")
	}
	if got != "ROOT/domain" {
		t.Errorf("expected %q, got %q", "ROOT/domain", got)
	}
}

// TestLogicalNameFromPath_SpecNodeDeep verifies a deeply nested spec node
// path maps correctly to ROOT/<path>.
func TestLogicalNameFromPath_SpecNodeDeep(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/tech_design/logical_names/_node.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/tech_design/logical_names/_node.md")
	}
	if got != "ROOT/tech_design/logical_names" {
		t.Errorf("expected %q, got %q", "ROOT/tech_design/logical_names", got)
	}
}

// TestLogicalNameFromPath_TestNodeRootCanonical verifies that the root-level
// canonical test file (spec/default.test.md) maps to the bare "TEST" logical
// name with no path suffix.
func TestLogicalNameFromPath_TestNodeRootCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/default.test.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/default.test.md")
	}
	if got != "TEST" {
		t.Errorf("expected %q, got %q", "TEST", got)
	}
}

// TestLogicalNameFromPath_TestNodeCanonical verifies the canonical test node
// (default.test.md) maps to TEST/<path> without a parenthesized qualifier.
func TestLogicalNameFromPath_TestNodeCanonical(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/domain/config/default.test.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/domain/config/default.test.md")
	}
	if got != "TEST/domain/config" {
		t.Errorf("expected %q, got %q", "TEST/domain/config", got)
	}
}

// TestLogicalNameFromPath_TestNodeNamed verifies a named test node maps to
// TEST/<path>(<name>) with the name in parentheses.
func TestLogicalNameFromPath_TestNodeNamed(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/domain/config/edge_cases.test.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/spec/domain/config/edge_cases.test.md")
	}
	if got != "TEST/domain/config(edge_cases)" {
		t.Errorf("expected %q, got %q", "TEST/domain/config(edge_cases)", got)
	}
}

// TestLogicalNameFromPath_ExternalDependency verifies an external dependency
// file maps to EXTERNAL/<name>.
func TestLogicalNameFromPath_ExternalDependency(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/external/celcoin-api/_external.md")
	if !ok {
		t.Fatal("expected ok=true for code-from-spec/external/celcoin-api/_external.md")
	}
	if got != "EXTERNAL/celcoin-api" {
		t.Errorf("expected %q, got %q", "EXTERNAL/celcoin-api", got)
	}
}

// TestLogicalNameFromPath_UnrecognizedPath verifies that a path that does not
// match any known pattern returns ("", false).
func TestLogicalNameFromPath_UnrecognizedPath(t *testing.T) {
	got, ok := LogicalNameFromPath("readme.md")
	if ok {
		t.Fatal("expected ok=false for readme.md")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestLogicalNameFromPath_PathWithoutNodeMd verifies that a .md file inside
// code-from-spec/spec/ that is not _node.md or .test.md returns ("", false).
func TestLogicalNameFromPath_PathWithoutNodeMd(t *testing.T) {
	got, ok := LogicalNameFromPath("code-from-spec/spec/domain/config/something.md")
	if ok {
		t.Fatal("expected ok=false for code-from-spec/spec/domain/config/something.md")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// PathFromLogicalName tests
// ---------------------------------------------------------------------------

// TestPathFromLogicalName_ROOT verifies that the bare "ROOT" logical name
// resolves to spec/_node.md.
func TestPathFromLogicalName_ROOT(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT")
	if !ok {
		t.Fatal("expected ok=true for ROOT")
	}
	if got != "code-from-spec/spec/_node.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/spec/_node.md", got)
	}
}

// TestPathFromLogicalName_ROOTWithPath verifies that ROOT/<path> resolves to
// spec/<path>/_node.md.
func TestPathFromLogicalName_ROOTWithPath(t *testing.T) {
	got, ok := PathFromLogicalName("ROOT/domain/staleness")
	if !ok {
		t.Fatal("expected ok=true for ROOT/domain/staleness")
	}
	if got != "code-from-spec/spec/domain/staleness/_node.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/spec/domain/staleness/_node.md", got)
	}
}

// TestPathFromLogicalName_TESTWithoutPath verifies that the bare "TEST" logical
// name (no path suffix) resolves to spec/default.test.md.
func TestPathFromLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := PathFromLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true for TEST")
	}
	if got != "code-from-spec/spec/default.test.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/spec/default.test.md", got)
	}
}

// TestPathFromLogicalName_TESTCanonical verifies that TEST/<path> (no
// parenthesized name) resolves to spec/<path>/default.test.md.
func TestPathFromLogicalName_TESTCanonical(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config")
	}
	if got != "code-from-spec/spec/domain/config/default.test.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/spec/domain/config/default.test.md", got)
	}
}

// TestPathFromLogicalName_TESTNamed verifies that TEST/<path>(<name>) resolves
// to spec/<path>/<name>.test.md.
func TestPathFromLogicalName_TESTNamed(t *testing.T) {
	got, ok := PathFromLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config(edge_cases)")
	}
	if got != "code-from-spec/spec/domain/config/edge_cases.test.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/spec/domain/config/edge_cases.test.md", got)
	}
}

// TestPathFromLogicalName_EXTERNAL verifies that EXTERNAL/<name> resolves to
// external/<name>/_external.md.
func TestPathFromLogicalName_EXTERNAL(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL/celcoin-api")
	if !ok {
		t.Fatal("expected ok=true for EXTERNAL/celcoin-api")
	}
	if got != "code-from-spec/external/celcoin-api/_external.md" {
		t.Errorf("expected %q, got %q", "code-from-spec/external/celcoin-api/_external.md", got)
	}
}

// TestPathFromLogicalName_UnrecognizedPrefix verifies that a logical name with
// an unknown prefix returns ("", false).
func TestPathFromLogicalName_UnrecognizedPrefix(t *testing.T) {
	got, ok := PathFromLogicalName("UNKNOWN/something")
	if ok {
		t.Fatal("expected ok=false for UNKNOWN/something")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestPathFromLogicalName_EmptyString verifies that an empty string returns
// ("", false).
func TestPathFromLogicalName_EmptyString(t *testing.T) {
	got, ok := PathFromLogicalName("")
	if ok {
		t.Fatal("expected ok=false for empty string")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestPathFromLogicalName_EXTERNALWithoutName verifies that bare "EXTERNAL"
// (no dependency name) returns ("", false) — a valid EXTERNAL logical name
// requires a name after the prefix.
func TestPathFromLogicalName_EXTERNALWithoutName(t *testing.T) {
	got, ok := PathFromLogicalName("EXTERNAL")
	if ok {
		t.Fatal("expected ok=false for EXTERNAL")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// LogicalNamesMatch tests
// ---------------------------------------------------------------------------

// TestLogicalNamesMatch_ExactMatch verifies that two identical logical names
// are considered equal.
func TestLogicalNamesMatch_ExactMatch(t *testing.T) {
	if !LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/config") {
		t.Error("expected ROOT/domain/config to match itself")
	}
}

// TestLogicalNamesMatch_DifferentNames verifies that two different logical
// names are not considered equal.
func TestLogicalNamesMatch_DifferentNames(t *testing.T) {
	if LogicalNamesMatch("ROOT/domain/config", "ROOT/domain/api") {
		t.Error("expected ROOT/domain/config and ROOT/domain/api to not match")
	}
}

// TestLogicalNamesMatch_TESTCanonicalVsDefault verifies that TEST/<path> and
// TEST/<path>(default) are considered equal — "default" is the canonical test
// name and both forms refer to the same test node.
func TestLogicalNamesMatch_TESTCanonicalVsDefault(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(default)") {
		t.Error("expected TEST/domain/config to match TEST/domain/config(default)")
	}
}

// TestLogicalNamesMatch_TESTDefaultVsCanonical verifies the symmetric case:
// TEST/<path>(default) matches TEST/<path>.
func TestLogicalNamesMatch_TESTDefaultVsCanonical(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(default)", "TEST/domain/config") {
		t.Error("expected TEST/domain/config(default) to match TEST/domain/config")
	}
}

// TestLogicalNamesMatch_TESTWithoutPathVsDefault verifies that bare "TEST" and
// "TEST(default)" are considered equal — both refer to the root-level canonical
// test node.
func TestLogicalNamesMatch_TESTWithoutPathVsDefault(t *testing.T) {
	if !LogicalNamesMatch("TEST", "TEST(default)") {
		t.Error("expected TEST to match TEST(default)")
	}
}

// TestLogicalNamesMatch_TESTNamedSameName verifies that two TEST nodes with
// the same parenthesized name are considered equal.
func TestLogicalNamesMatch_TESTNamedSameName(t *testing.T) {
	if !LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(edge_cases)") {
		t.Error("expected TEST/domain/config(edge_cases) to match itself")
	}
}

// TestLogicalNamesMatch_TESTNamedDifferentName verifies that two TEST nodes
// with different parenthesized names are not considered equal.
func TestLogicalNamesMatch_TESTNamedDifferentName(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config(edge_cases)", "TEST/domain/config(smoke)") {
		t.Error("expected TEST/domain/config(edge_cases) and TEST/domain/config(smoke) to not match")
	}
}

// TestLogicalNamesMatch_TESTCanonicalVsNamed verifies that the canonical form
// TEST/<path> does NOT match a non-default named form TEST/<path>(<name>).
func TestLogicalNamesMatch_TESTCanonicalVsNamed(t *testing.T) {
	if LogicalNamesMatch("TEST/domain/config", "TEST/domain/config(edge_cases)") {
		t.Error("expected TEST/domain/config and TEST/domain/config(edge_cases) to not match")
	}
}

// ---------------------------------------------------------------------------
// HasParent tests
// ---------------------------------------------------------------------------

// TestHasParent_ROOT verifies that the root node has no parent but is a valid
// logical name (hasParent=false, ok=true).
func TestHasParent_ROOT(t *testing.T) {
	hasParent, ok := HasParent("ROOT")
	if !ok {
		t.Fatal("expected ok=true for ROOT")
	}
	if hasParent {
		t.Error("expected hasParent=false for ROOT")
	}
}

// TestHasParent_ROOTWithPath verifies that a ROOT node below the root has a
// parent (hasParent=true, ok=true).
func TestHasParent_ROOTWithPath(t *testing.T) {
	hasParent, ok := HasParent("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true for ROOT/domain/config")
	}
	if !hasParent {
		t.Error("expected hasParent=true for ROOT/domain/config")
	}
}

// TestHasParent_TESTWithoutPath verifies that bare "TEST" has a parent
// (its parent is ROOT).
func TestHasParent_TESTWithoutPath(t *testing.T) {
	hasParent, ok := HasParent("TEST")
	if !ok {
		t.Fatal("expected ok=true for TEST")
	}
	if !hasParent {
		t.Error("expected hasParent=true for TEST")
	}
}

// TestHasParent_TESTWithPath verifies that TEST/<path> has a parent
// (parent is in the ROOT namespace).
func TestHasParent_TESTWithPath(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config")
	}
	if !hasParent {
		t.Error("expected hasParent=true for TEST/domain/config")
	}
}

// TestHasParent_TESTNamed verifies that TEST/<path>(<name>) has a parent
// (parent is in the ROOT namespace).
func TestHasParent_TESTNamed(t *testing.T) {
	hasParent, ok := HasParent("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config(edge_cases)")
	}
	if !hasParent {
		t.Error("expected hasParent=true for TEST/domain/config(edge_cases)")
	}
}

// TestHasParent_EXTERNAL verifies that EXTERNAL/<name> has no parent but is a
// valid logical name (hasParent=false, ok=true).
func TestHasParent_EXTERNAL(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL/celcoin-api")
	if !ok {
		t.Fatal("expected ok=true for EXTERNAL/celcoin-api")
	}
	if hasParent {
		t.Error("expected hasParent=false for EXTERNAL/celcoin-api")
	}
}

// TestHasParent_EXTERNALWithoutName verifies that bare "EXTERNAL" (no
// dependency name) is not a valid logical name (ok=false).
func TestHasParent_EXTERNALWithoutName(t *testing.T) {
	hasParent, ok := HasParent("EXTERNAL")
	if ok {
		t.Fatal("expected ok=false for EXTERNAL")
	}
	if hasParent {
		t.Error("expected hasParent=false for EXTERNAL")
	}
}

// TestHasParent_EmptyString verifies that an empty string is not a valid
// logical name (ok=false).
func TestHasParent_EmptyString(t *testing.T) {
	hasParent, ok := HasParent("")
	if ok {
		t.Fatal("expected ok=false for empty string")
	}
	if hasParent {
		t.Error("expected hasParent=false for empty string")
	}
}

// TestHasParent_UnrecognizedPrefix verifies that a logical name with an
// unknown prefix is not valid (ok=false).
func TestHasParent_UnrecognizedPrefix(t *testing.T) {
	hasParent, ok := HasParent("UNKNOWN/something")
	if ok {
		t.Fatal("expected ok=false for UNKNOWN/something")
	}
	if hasParent {
		t.Error("expected hasParent=false for UNKNOWN/something")
	}
}

// ---------------------------------------------------------------------------
// ParentLogicalName tests
// ---------------------------------------------------------------------------

// TestParentLogicalName_ROOTSingleSegment verifies that ROOT/<single> has
// parent ROOT.
func TestParentLogicalName_ROOTSingleSegment(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain")
	if !ok {
		t.Fatal("expected ok=true for ROOT/domain")
	}
	if got != "ROOT" {
		t.Errorf("expected %q, got %q", "ROOT", got)
	}
}

// TestParentLogicalName_ROOTTwoSegments verifies that ROOT/x/y has parent
// ROOT/x (strip last segment).
func TestParentLogicalName_ROOTTwoSegments(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/domain/config")
	if !ok {
		t.Fatal("expected ok=true for ROOT/domain/config")
	}
	if got != "ROOT/domain" {
		t.Errorf("expected %q, got %q", "ROOT/domain", got)
	}
}

// TestParentLogicalName_ROOTThreeSegments verifies that ROOT/x/y/z has parent
// ROOT/x/y (strip last segment).
func TestParentLogicalName_ROOTThreeSegments(t *testing.T) {
	got, ok := ParentLogicalName("ROOT/tech_design/logical_names")
	if !ok {
		t.Fatal("expected ok=true for ROOT/tech_design/logical_names")
	}
	if got != "ROOT/tech_design" {
		t.Errorf("expected %q, got %q", "ROOT/tech_design", got)
	}
}

// TestParentLogicalName_TESTWithoutPath verifies that bare "TEST" has parent
// "ROOT" — the root-level test node's parent is the root spec node.
func TestParentLogicalName_TESTWithoutPath(t *testing.T) {
	got, ok := ParentLogicalName("TEST")
	if !ok {
		t.Fatal("expected ok=true for TEST")
	}
	if got != "ROOT" {
		t.Errorf("expected %q, got %q", "ROOT", got)
	}
}

// TestParentLogicalName_TESTWithPath verifies that TEST/<path> has parent
// ROOT/<path> — test nodes' parents are in the ROOT namespace.
func TestParentLogicalName_TESTWithPath(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config")
	}
	if got != "ROOT/domain/config" {
		t.Errorf("expected %q, got %q", "ROOT/domain/config", got)
	}
}

// TestParentLogicalName_TESTNamed verifies that TEST/<path>(<name>) has parent
// ROOT/<path> — the parenthesized name is stripped and the parent is in the
// ROOT namespace.
func TestParentLogicalName_TESTNamed(t *testing.T) {
	got, ok := ParentLogicalName("TEST/domain/config(edge_cases)")
	if !ok {
		t.Fatal("expected ok=true for TEST/domain/config(edge_cases)")
	}
	if got != "ROOT/domain/config" {
		t.Errorf("expected %q, got %q", "ROOT/domain/config", got)
	}
}

// TestParentLogicalName_ROOTHasNoParent verifies that bare "ROOT" returns
// ("", false) — the root node has no parent.
func TestParentLogicalName_ROOTHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("ROOT")
	if ok {
		t.Fatal("expected ok=false for ROOT")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestParentLogicalName_EXTERNALHasNoParent verifies that EXTERNAL/<name>
// returns ("", false) — external dependencies have no parent.
func TestParentLogicalName_EXTERNALHasNoParent(t *testing.T) {
	got, ok := ParentLogicalName("EXTERNAL/celcoin-api")
	if ok {
		t.Fatal("expected ok=false for EXTERNAL/celcoin-api")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// TestParentLogicalName_InvalidInput verifies that an empty string returns
// ("", false) — not a valid logical name.
func TestParentLogicalName_InvalidInput(t *testing.T) {
	got, ok := ParentLogicalName("")
	if ok {
		t.Fatal("expected ok=false for empty string")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
