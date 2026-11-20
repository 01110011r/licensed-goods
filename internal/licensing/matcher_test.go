package licensing

import (
	"reflect"
	"strings"
	"testing"
)

// ikpu pads a prefix with zeros to a valid 17-digit code.
func ikpu(prefix string) string {
	return prefix + strings.Repeat("0", IKPULength-len(prefix))
}

func fixtureCategories() map[string]Category {
	return map[string]Category{
		"c1":  {ID: "c1"},
		"c2":  {ID: "c2"},                 // Medicines (root)
		"c3":  {ID: "c3", ParentID: "c2"}, // Painkillers
		"c4":  {ID: "c4", ParentID: "c1"}, // Tea
		"c8":  {ID: "c8"},                 // Alcohol
		"c9":  {ID: "c9"},                 // Juice
		"c11": {ID: "c11"},                // Pesticides
	}
}

func fixtureGroups() []Group {
	return []Group{
		{ID: "MED", Rules: []Rule{
			{Kind: RuleCategory, CategoryID: "c2"},
			{Kind: RuleIKPU, IKPUPrefix: "30049"},
			{Kind: RuleIKPU, IKPUPrefix: "300490", Excluded: true},
		}},
		{ID: "ALC", Rules: []Rule{
			{Kind: RuleCategory, CategoryID: "c8"},
			{Kind: RuleIKPU, IKPUPrefix: "2204"},
			{Kind: RuleIKPU, IKPUPrefix: "2208"},
		}},
		{ID: "PEST", Rules: []Rule{
			{Kind: RuleCategory, CategoryID: "c11"},
			{Kind: RuleIKPU, IKPUPrefix: "3808"},
		}},
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name string
		p    Product
		want []string
	}{
		{"category descendant matches (p1)", Product{"p1", "c3", ikpu("300491")}, []string{"MED"}},
		{"longest prefix is exclusion -> no MED (p4)", Product{"p4", "c4", ikpu("300490")}, nil},
		{"category + excluded code still MED (p11)", Product{"p11", "c3", ikpu("300490")}, []string{"MED"}},
		{"code matches, category doesn't (ALC, p7)", Product{"p7", "c9", ikpu("2208")}, []string{"ALC"}},
		{"category matches, code doesn't (p12)", Product{"p12", "c8", ikpu("2209")}, []string{"ALC"}},
		{"category and code both match (p5)", Product{"p5", "c8", ikpu("2204")}, []string{"ALC"}},
		{"pesticide (p8)", Product{"p8", "c11", ikpu("3808")}, []string{"PEST"}},
		{"code matches MED only", Product{"x", "c1", ikpu("300491")}, []string{"MED"}},
		{"multiple groups match, in group order", Product{"x", "c3", ikpu("2208")}, []string{"MED", "ALC"}},
		{"invalid short code: no code match (p14)", Product{"p14", "c4", "3004"}, nil},
		{"invalid code but category still matches (p14)", Product{"p14", "c3", "3004"}, []string{"MED"}},
		{"non-digit code is invalid", Product{"x", "c4", "30049abcdefghijkl"}, nil},
		{"unknown category, code still matches (p13)", Product{"p13", "c99", ikpu("2204")}, []string{"ALC"}},
		{"unknown category, nothing else", Product{"p13", "c99", ikpu("1111")}, nil},
		{"no matches at all", Product{"x", "c1", ikpu("1111")}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Match(tc.p, fixtureCategories(), fixtureGroups()).GroupIDs
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMatch_ExclusionTieWins(t *testing.T) {
	groups := []Group{{ID: "G", Rules: []Rule{
		{Kind: RuleIKPU, IKPUPrefix: "1234"},
		{Kind: RuleIKPU, IKPUPrefix: "1234", Excluded: true},
	}}}
	got := Match(Product{"x", "c1", ikpu("1234")}, fixtureCategories(), groups).GroupIDs
	if len(got) != 0 {
		t.Fatalf("expected exclusion to win on tie, got %v", got)
	}
}

func TestMatch_ExclusionDoesNotAffectOtherGroups(t *testing.T) {
	// MED excludes 300490, but another group may still match that code.
	groups := append(fixtureGroups(), Group{ID: "OTHER", Rules: []Rule{
		{Kind: RuleIKPU, IKPUPrefix: "30"},
	}})
	got := Match(Product{"x", "c4", ikpu("300490")}, fixtureCategories(), groups).GroupIDs
	if !reflect.DeepEqual(got, []string{"OTHER"}) {
		t.Fatalf("got %v, want [OTHER]", got)
	}
}

func TestMatch_Issues(t *testing.T) {
	res := Match(Product{"p13", "c99", "123"}, fixtureCategories(), fixtureGroups())
	kinds := map[IssueKind]bool{}
	for _, i := range res.Issues {
		kinds[i.Kind] = true
	}
	if !kinds[IssueUnknownCategory] || !kinds[IssueInvalidIKPU] {
		t.Fatalf("missing expected issues: %+v", res.Issues)
	}
}

func TestMatch_MissingParent(t *testing.T) {
	cats := map[string]Category{
		"a": {ID: "a", ParentID: "ghost"},
	}
	groups := []Group{{ID: "G", Rules: []Rule{{Kind: RuleCategory, CategoryID: "a"}}}}
	res := Match(Product{"x", "a", ikpu("1")}, cats, groups)
	if !reflect.DeepEqual(res.GroupIDs, []string{"G"}) {
		t.Fatalf("own category should still match, got %v", res.GroupIDs)
	}
	if len(res.Issues) != 1 || res.Issues[0].Kind != IssueMissingParent {
		t.Fatalf("expected missing_parent issue, got %+v", res.Issues)
	}
}

func TestMatch_CyclicHierarchy(t *testing.T) {
	cats := map[string]Category{
		"a": {ID: "a", ParentID: "b"},
		"b": {ID: "b", ParentID: "c"},
		"c": {ID: "c", ParentID: "a"}, // a -> b -> c -> a
		"s": {ID: "s", ParentID: "s"}, // self-loop
	}
	groups := []Group{{ID: "G", Rules: []Rule{{Kind: RuleCategory, CategoryID: "c"}}}}

	res := Match(Product{"x", "a", ikpu("1")}, cats, groups) // must terminate
	if !reflect.DeepEqual(res.GroupIDs, []string{"G"}) {
		t.Fatalf("ancestor in cycle should still match, got %v", res.GroupIDs)
	}
	hasCycle := false
	for _, i := range res.Issues {
		hasCycle = hasCycle || i.Kind == IssueCategoryCycle
	}
	if !hasCycle {
		t.Fatalf("expected cycle issue, got %+v", res.Issues)
	}

	// self-loop terminates too
	Match(Product{"y", "s", ikpu("1")}, cats, groups)
}

func TestMatch_EmptyInputs(t *testing.T) {
	res := Match(Product{}, nil, nil)
	if len(res.GroupIDs) != 0 {
		t.Fatalf("expected no groups, got %v", res.GroupIDs)
	}
}

func TestMatch_IgnoresExcludedCategoryRuleAndEmptyPrefix(t *testing.T) {
	groups := []Group{{ID: "G", Rules: []Rule{
		{Kind: RuleCategory, CategoryID: "c3", Excluded: true},
		{Kind: RuleIKPU, IKPUPrefix: ""}, // would match everything if not skipped
	}}}
	got := Match(Product{"x", "c3", ikpu("1")}, fixtureCategories(), groups).GroupIDs
	if len(got) != 0 {
		t.Fatalf("got %v, want none", got)
	}
}
