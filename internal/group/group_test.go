package group

import "testing"

func TestGroupTypeValid(t *testing.T) {
	validTypes := []GroupType{
		GroupTypeSelect,
		GroupTypeURLTest,
		GroupTypeFallback,
	}

	for _, groupType := range validTypes {
		if !groupType.Valid() {
			t.Fatalf("%q should be valid", groupType)
		}
	}

	if GroupType("auto-stable").Valid() {
		t.Fatal("auto-stable should not be valid in Phase 1")
	}
}

func TestProxyGroupValidate(t *testing.T) {
	group := ProxyGroup{
		Name:    "PROXY",
		Type:    GroupTypeSelect,
		Proxies: []string{"DIRECT", "node-a"},
	}

	if err := group.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	group.Name = ""
	if err := group.Validate(); err != ErrEmptyGroupName {
		t.Fatalf("Validate() error = %v, want %v", err, ErrEmptyGroupName)
	}

	group.Name = "AUTO-STABLE"
	group.Type = GroupType("auto-stable")
	if err := group.Validate(); err != ErrUnsupportedGroupType {
		t.Fatalf("Validate() error = %v, want %v", err, ErrUnsupportedGroupType)
	}
}
