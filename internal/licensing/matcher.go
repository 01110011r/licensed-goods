package licensing

import "strings"

// IKPULength is the number of digits a valid IKPU code must have.
const IKPULength = 17

// RuleKind says which field of a Rule is meaningful.
type RuleKind string

const (
	RuleCategory RuleKind = "category"
	RuleIKPU     RuleKind = "ikpu"
)

// Category is a node in the category tree. ParentID is "" for roots.
type Category struct {
	ID       string
	ParentID string
}

// Product holds only the fields the matcher needs.
type Product struct {
	ID         string
	CategoryID string
	IKPU       string
}

// Rule belongs to a licence group.
//   - RuleCategory: CategoryID is set; the rule covers that category and all descendants.
//   - RuleIKPU:     IKPUPrefix is set; Excluded marks an exclusion prefix.
type Rule struct {
	Kind       RuleKind
	CategoryID string
	IKPUPrefix string
	Excluded   bool
}

// Group is a licence group (MED, ALC, PEST, ...) with its rules.
type Group struct {
	ID    string
	Rules []Rule
}

// IssueKind describes a data problem found while matching one product.
type IssueKind string

const (
	IssueUnknownCategory IssueKind = "unknown_category"
	IssueCategoryCycle   IssueKind = "category_cycle"
	IssueMissingParent   IssueKind = "missing_parent"
	IssueInvalidIKPU     IssueKind = "invalid_ikpu"
)

// Issue is a non-fatal data problem. Matching continues despite it.
type Issue struct {
	Kind   IssueKind `json:"kind"`
	Detail string    `json:"detail"`
}

// Result is the outcome of matching one product.
type Result struct {
	GroupIDs []string // required groups, in the order of the input groups
	Issues   []Issue
}

// Match returns every licence group the product requires.
// It never panics and never loops forever on malformed data; a malformed
// field only disables the matching path that depends on it.
func Match(p Product, categories map[string]Category, groups []Group) Result {
	var res Result

	ancestors, catIssues := categoryChain(p.CategoryID, categories)
	res.Issues = append(res.Issues, catIssues...)

	codeValid := isValidIKPU(p.IKPU)
	if !codeValid {
		res.Issues = append(res.Issues, Issue{IssueInvalidIKPU, "ikpu=" + p.IKPU})
	}

	for _, g := range groups {
		if matchesCategory(ancestors, g.Rules) || (codeValid && matchesIKPU(p.IKPU, g.Rules)) {
			res.GroupIDs = append(res.GroupIDs, g.ID)
		}
	}
	return res
}

// categoryChain returns the set containing the category and all its ancestors.
func categoryChain(categoryID string, cats map[string]Category) (map[string]struct{}, []Issue) {
	chain := map[string]struct{}{}
	var issues []Issue

	if _, ok := cats[categoryID]; !ok {
		return chain, []Issue{{IssueUnknownCategory, "category=" + categoryID}}
	}

	cur := categoryID
	for cur != "" {
		if _, seen := chain[cur]; seen {
			issues = append(issues, Issue{IssueCategoryCycle, "at=" + cur})
			break
		}
		c, ok := cats[cur]
		if !ok {
			issues = append(issues, Issue{IssueMissingParent, "parent=" + cur})
			break
		}
		chain[cur] = struct{}{}
		cur = c.ParentID
	}
	return chain, issues
}

func matchesCategory(chain map[string]struct{}, rules []Rule) bool {
	for _, r := range rules {
		if r.Kind != RuleCategory || r.Excluded {
			continue
		}
		if _, ok := chain[r.CategoryID]; ok {
			return true
		}
	}
	return false
}

// matchesIKPU applies longest-prefix-wins. On an equal-length tie,
// the exclusion wins (the safer reading).
func matchesIKPU(code string, rules []Rule) bool {
	bestLen := -1
	bestExcluded := false
	for _, r := range rules {
		if r.Kind != RuleIKPU || r.IKPUPrefix == "" {
			continue
		}
		if !strings.HasPrefix(code, r.IKPUPrefix) {
			continue
		}
		n := len(r.IKPUPrefix)
		switch {
		case n > bestLen:
			bestLen, bestExcluded = n, r.Excluded
		case n == bestLen && r.Excluded:
			bestExcluded = true
		}
	}
	return bestLen >= 0 && !bestExcluded
}

func isValidIKPU(code string) bool {
	if len(code) != IKPULength {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}
