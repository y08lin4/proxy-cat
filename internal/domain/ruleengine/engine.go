package ruleengine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

type Engine struct{}

func New() *Engine { return &Engine{} }

// Validate checks a rule and returns a list of issues (empty = valid).
func (e *Engine) Validate(rule proxy.Rule) []ValidationError {
	var errs []ValidationError
	if strings.TrimSpace(rule.Type) == "" {
		errs = append(errs, ValidationError{RuleID: rule.ID, Field: "type", Message: "rule type is required"})
	}
	if !isLogicalRule(rule.Type) && strings.EqualFold(rule.Type, "MATCH") {
		// MATCH can have empty value
	} else if !isLogicalRule(rule.Type) && !isNoValueType(rule.Type) && strings.TrimSpace(rule.Value) == "" {
		errs = append(errs, ValidationError{RuleID: rule.ID, Field: "value", Message: "rule value is required"})
	}
	// Validate sub-rules for logical rules
	if isLogicalRule(rule.Type) && len(rule.SubRules) == 0 {
		errs = append(errs, ValidationError{RuleID: rule.ID, Field: "subRules", Message: "logical rule requires sub-rules"})
	}
	return errs
}

// ToMihomoRule converts a rule to Mihomo YAML rule line(s).
func (e *Engine) ToMihomoRule(rule proxy.Rule) ([]string, error) {
	if !rule.Enabled {
		return nil, nil
	}
	if rule.InlineRule != "" {
		return []string{rule.InlineRule}, nil
	}
	if isLogicalRule(rule.Type) {
		return e.flattenLogicalRule(rule)
	}
	return e.flattenLeafRule(rule), nil
}

func (e *Engine) flattenLeafRule(rule proxy.Rule) []string {
	typ := strings.TrimSpace(rule.Type)
	value := strings.TrimSpace(rule.Value)
	target := strings.TrimSpace(rule.TargetGroup)
	if strings.EqualFold(typ, "MATCH") || value == "" {
		return []string{fmt.Sprintf("%s,%s", typ, target)}
	}
	return []string{fmt.Sprintf("%s,%s,%s", typ, value, target)}
}

func (e *Engine) flattenLogicalRule(rule proxy.Rule) ([]string, error) {
	var parts []string
	for _, sub := range rule.SubRules {
		subLines, err := e.ToMihomoRule(sub)
		if err != nil {
			return nil, err
		}
		for _, line := range subLines {
			parts = append(parts, fmt.Sprintf("(%s)", line))
		}
	}
	target := strings.TrimSpace(rule.TargetGroup)
	return []string{fmt.Sprintf("%s,(%s),%s", strings.ToUpper(rule.Type), strings.Join(parts, ","), target)}, nil
}

func (e *Engine) SortRules(rules []proxy.Rule) []proxy.Rule {
	sorted := append([]proxy.Rule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Priority < sorted[j].Priority })
	return sorted
}

func (e *Engine) ReorderRules(rules []proxy.Rule, spacing int) []proxy.Rule {
	result := append([]proxy.Rule(nil), rules...)
	for i := range result {
		result[i].Priority = i * spacing
	}
	return result
}

type ValidationError struct {
	RuleID  string `json:"ruleId"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("rule %s: %s: %s", ve.RuleID, ve.Field, ve.Message)
}

func isLogicalRule(typ string) bool {
	switch strings.ToUpper(typ) {
	case "AND", "OR", "NOT":
		return true
	}
	return false
}

func isNoValueType(typ string) bool {
	return strings.EqualFold(typ, "MATCH")
}
