package group

import "errors"

// GroupType is the proxy group behavior supported by Proxy-Cat.
type GroupType string

const (
	GroupTypeSelect   GroupType = "select"
	GroupTypeURLTest  GroupType = "url-test"
	GroupTypeFallback GroupType = "fallback"
)

var ErrEmptyGroupName = errors.New("group name is empty")

// ProxyGroup is the minimal internal representation needed for Phase 1.
type ProxyGroup struct {
	Name          string
	Type          GroupType
	Proxies       []string
	SelectedProxy string
	URL           string
	IntervalSec   int
}

// Valid reports whether the group type is supported in Phase 1.
func (t GroupType) Valid() bool {
	switch t {
	case GroupTypeSelect, GroupTypeURLTest, GroupTypeFallback:
		return true
	default:
		return false
	}
}

// Validate checks the minimal invariants shared by all Phase 1 group types.
func (g ProxyGroup) Validate() error {
	if g.Name == "" {
		return ErrEmptyGroupName
	}
	if !g.Type.Valid() {
		return ErrUnsupportedGroupType
	}
	return nil
}
