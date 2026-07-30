package ruleengine

import (
	"testing"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

func toLines(t *testing.T, rule proxy.Rule) []string {
	t.Helper()
	lines, err := New().ToMihomoRule(rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return lines
}

// ---- Leaf rules ----

func TestToMihomoRule_Domain(t *testing.T) {
	rule := proxy.Rule{ID: "1", Type: "DOMAIN", Value: "example.com", TargetGroup: "PROXY", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "DOMAIN,example.com,PROXY" {
		t.Fatalf("expected 'DOMAIN,example.com,PROXY', got %v", lines)
	}
}

func TestToMihomoRule_DomainSuffix(t *testing.T) {
	rule := proxy.Rule{ID: "2", Type: "DOMAIN-SUFFIX", Value: "google.com", TargetGroup: "PROXY", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "DOMAIN-SUFFIX,google.com,PROXY" {
		t.Fatalf("expected 'DOMAIN-SUFFIX,google.com,PROXY', got %v", lines)
	}
}

func TestToMihomoRule_GeoIP(t *testing.T) {
	rule := proxy.Rule{ID: "3", Type: "GEOIP", Value: "CN", TargetGroup: "DIRECT", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "GEOIP,CN,DIRECT" {
		t.Fatalf("expected 'GEOIP,CN,DIRECT', got %v", lines)
	}
}

func TestToMihomoRule_Match(t *testing.T) {
	rule := proxy.Rule{ID: "4", Type: "MATCH", TargetGroup: "PROXY", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "MATCH,PROXY" {
		t.Fatalf("expected 'MATCH,PROXY', got %v", lines)
	}
}

func TestToMihomoRule_MatchWithExtraValue(t *testing.T) {
	// Even if Value is set, MATCH ignores it (no value type).
	rule := proxy.Rule{ID: "5", Type: "MATCH", Value: "ignored", TargetGroup: "PROXY", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "MATCH,PROXY" {
		t.Fatalf("expected 'MATCH,PROXY', got %v", lines)
	}
}

func TestToMihomoRule_EmptyValue_DomainKeyword(t *testing.T) {
	// DOMAIN-KEYWORD with empty value should output type,target only.
	rule := proxy.Rule{ID: "6", Type: "DOMAIN-KEYWORD", Value: "", TargetGroup: "DIRECT", Enabled: true}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "DOMAIN-KEYWORD,DIRECT" {
		t.Fatalf("expected 'DOMAIN-KEYWORD,DIRECT', got %v", lines)
	}
}

// ---- Inline rules ----

func TestToMihomoRule_Inline(t *testing.T) {
	rule := proxy.Rule{ID: "7", Type: "DOMAIN-SUFFIX", Value: "x.com", TargetGroup: "PROXY", Enabled: true, InlineRule: "DOMAIN-SUFFIX,x.com,DIRECT"}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "DOMAIN-SUFFIX,x.com,DIRECT" {
		t.Fatalf("expected inline rule to be used, got %v", lines)
	}
}

// ---- Logical rules ----

func TestToMihomoRule_AND(t *testing.T) {
	rule := proxy.Rule{
		ID:   "10",
		Type: "AND",
		SubRules: []proxy.Rule{
			{ID: "10a", Type: "DOMAIN", Value: "a.com", TargetGroup: "PROXY", Enabled: true},
			{ID: "10b", Type: "DOMAIN", Value: "b.com", TargetGroup: "PROXY", Enabled: true},
		},
		TargetGroup: "PROXY",
		Enabled:     true,
	}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "AND,((DOMAIN,a.com,PROXY),(DOMAIN,b.com,PROXY)),PROXY" {
		t.Fatalf("unexpected AND output: %v", lines)
	}
}

func TestToMihomoRule_OR(t *testing.T) {
	rule := proxy.Rule{
		ID:   "11",
		Type: "OR",
		SubRules: []proxy.Rule{
			{ID: "11a", Type: "DOMAIN-SUFFIX", Value: "x.com", TargetGroup: "DIRECT", Enabled: true},
			{ID: "11b", Type: "DOMAIN-SUFFIX", Value: "y.com", TargetGroup: "DIRECT", Enabled: true},
		},
		TargetGroup: "DIRECT",
		Enabled:     true,
	}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "OR,((DOMAIN-SUFFIX,x.com,DIRECT),(DOMAIN-SUFFIX,y.com,DIRECT)),DIRECT" {
		t.Fatalf("unexpected OR output: %v", lines)
	}
}

func TestToMihomoRule_NOT(t *testing.T) {
	rule := proxy.Rule{
		ID:   "12",
		Type: "NOT",
		SubRules: []proxy.Rule{
			{ID: "12a", Type: "GEOIP", Value: "CN", TargetGroup: "DIRECT", Enabled: true},
		},
		TargetGroup: "PROXY",
		Enabled:     true,
	}
	lines := toLines(t, rule)
	if len(lines) != 1 || lines[0] != "NOT,((GEOIP,CN,DIRECT)),PROXY" {
		t.Fatalf("unexpected NOT output: %v", lines)
	}
}

func TestToMihomoRule_NestedLogical(t *testing.T) {
	rule := proxy.Rule{
		ID:   "13",
		Type: "OR",
		SubRules: []proxy.Rule{
			{
				ID:   "13-inner",
				Type: "AND",
				SubRules: []proxy.Rule{
					{ID: "13a", Type: "DOMAIN", Value: "a.com", TargetGroup: "PROXY", Enabled: true},
					{ID: "13b", Type: "DOMAIN", Value: "b.com", TargetGroup: "PROXY", Enabled: true},
				},
				TargetGroup: "PROXY",
				Enabled:     true,
			},
			{ID: "13c", Type: "DOMAIN-SUFFIX", Value: "c.com", TargetGroup: "DIRECT", Enabled: true},
		},
		TargetGroup: "PROXY",
		Enabled:     true,
	}
	lines := toLines(t, rule)
	expected := "OR,((AND,((DOMAIN,a.com,PROXY),(DOMAIN,b.com,PROXY)),PROXY),(DOMAIN-SUFFIX,c.com,DIRECT)),PROXY"
	if len(lines) != 1 || lines[0] != expected {
		t.Fatalf("expected %q, got %v", expected, lines)
	}
}

// ---- Disabled rules ----

func TestToMihomoRule_Disabled(t *testing.T) {
	rule := proxy.Rule{ID: "20", Type: "DOMAIN", Value: "example.com", TargetGroup: "PROXY", Enabled: false}
	lines, err := New().ToMihomoRule(rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected no lines for disabled rule, got %v", lines)
	}
}

func TestToMihomoRule_DisabledLogical(t *testing.T) {
	rule := proxy.Rule{
		ID:   "21",
		Type: "AND",
		SubRules: []proxy.Rule{
			{ID: "21a", Type: "DOMAIN", Value: "a.com", TargetGroup: "PROXY", Enabled: true},
		},
		TargetGroup: "PROXY",
		Enabled:     false,
	}
	lines, err := New().ToMihomoRule(rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("expected no lines for disabled logical rule, got %v", lines)
	}
}

// ---- Validation ----

func TestValidate_Valid(t *testing.T) {
	rule := proxy.Rule{ID: "v1", Type: "DOMAIN", Value: "example.com", TargetGroup: "PROXY", Enabled: true}
	errs := New().Validate(rule)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestValidate_EmptyType(t *testing.T) {
	rule := proxy.Rule{ID: "v2", Type: "", Value: "example.com", TargetGroup: "PROXY", Enabled: true}
	errs := New().Validate(rule)
	if len(errs) != 1 || errs[0].Field != "type" {
		t.Fatalf("expected type error, got %v", errs)
	}
}

func TestValidate_MissingValue(t *testing.T) {
	rule := proxy.Rule{ID: "v3", Type: "DOMAIN", Value: "", TargetGroup: "PROXY", Enabled: true}
	errs := New().Validate(rule)
	if len(errs) != 1 || errs[0].Field != "value" {
		t.Fatalf("expected value error, got %v", errs)
	}
}

func TestValidate_MatchNoValueOk(t *testing.T) {
	rule := proxy.Rule{ID: "v4", Type: "MATCH", Value: "", TargetGroup: "PROXY", Enabled: true}
	errs := New().Validate(rule)
	if len(errs) != 0 {
		t.Fatalf("expected no errors for MATCH without value, got %v", errs)
	}
}

func TestValidate_LogicalNoSubRules(t *testing.T) {
	rule := proxy.Rule{ID: "v5", Type: "AND", Value: "", TargetGroup: "PROXY", Enabled: true}
	errs := New().Validate(rule)
	if len(errs) != 1 || errs[0].Field != "subRules" {
		t.Fatalf("expected subRules error, got %v", errs)
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := ValidationError{RuleID: "r1", Field: "type", Message: "bad"}
	if s := ve.Error(); s != "rule r1: type: bad" {
		t.Fatalf("unexpected Error() string: %s", s)
	}
}

// ---- Sort and Reorder ----

func TestSortRules(t *testing.T) {
	rules := []proxy.Rule{
		{ID: "a", Priority: 30},
		{ID: "b", Priority: 10},
		{ID: "c", Priority: 20},
	}
	sorted := New().SortRules(rules)
	if sorted[0].Priority != 10 || sorted[1].Priority != 20 || sorted[2].Priority != 30 {
		t.Fatalf("rules not sorted by priority: %+v", sorted)
	}
	// Original should be unchanged.
	if rules[0].Priority != 30 {
		t.Fatalf("original slice was mutated")
	}
}

func TestReorderRules(t *testing.T) {
	rules := []proxy.Rule{
		{ID: "x"},
		{ID: "y"},
		{ID: "z"},
	}
	result := New().ReorderRules(rules, 10)
	for i, r := range result {
		if r.Priority != i*10 {
			t.Fatalf("expected priority %d, got %d for rule %s", i*10, r.Priority, r.ID)
		}
	}
	// Original should have zero priorities.
	for i, r := range rules {
		if r.Priority != 0 {
			t.Fatalf("original slice mutated at index %d: %d", i, r.Priority)
		}
	}
}
