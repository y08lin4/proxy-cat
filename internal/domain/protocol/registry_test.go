package protocol

import "testing"

func TestBuildDefaultRegistry_HasAll13Protocols(t *testing.T) {
	r := BuildDefaultRegistry()
	all := r.All()
	if len(all) != 13 {
		t.Fatalf("expected 13 protocols, got %d", len(all))
	}
}

func TestGet_ReturnsCorrectProtocolByType(t *testing.T) {
	r := BuildDefaultRegistry()

	def, ok := r.Get("ss")
	if !ok {
		t.Fatal("expected ss protocol to be found")
	}
	if def.Name != "Shadowsocks" {
		t.Errorf("expected Shadowsocks, got %s", def.Name)
	}
	if len(def.Fields) != 6 {
		t.Errorf("expected 6 fields for ss, got %d", len(def.Fields))
	}

	def, ok = r.Get("vmess")
	if !ok {
		t.Fatal("expected vmess protocol to be found")
	}
	if def.Name != "VMess" {
		t.Errorf("expected VMess, got %s", def.Name)
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent protocol to not be found")
	}
}

func TestAll_ReturnsProtocolsInRegistrationOrder(t *testing.T) {
	r := BuildDefaultRegistry()
	all := r.All()

	expectedOrder := []string{
		"ss", "vmess", "trojan", "hysteria2", "vless",
		"socks5", "http", "hysteria", "tuic", "wireguard",
		"snell", "ssr", "ssh",
	}

	for i, exp := range expectedOrder {
		if all[i].Type != exp {
			t.Errorf("index %d: expected %s, got %s", i, exp, all[i].Type)
		}
	}
}

func TestNew_CreatesEmptyRegistry(t *testing.T) {
	r := New()
	if len(r.All()) != 0 {
		t.Errorf("expected empty registry, got %d protocols", len(r.All()))
	}
	if _, ok := r.Get("ss"); ok {
		t.Error("expected Get on empty registry to return false")
	}
}

func TestRegister_ThenAll_IncludesProtocol(t *testing.T) {
	r := New()
	r.Register(ProtocolDef{
		Type: "test", Name: "Test", Description: "Test protocol",
		Transport: []string{"tcp"},
		Fields:    []FieldDef{{Name: "addr", Type: "string", Required: true}},
	})
	all := r.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(all))
	}
	def, ok := r.Get("test")
	if !ok {
		t.Fatal("expected test protocol to be found")
	}
	if def.Name != "Test" {
		t.Errorf("expected Test, got %s", def.Name)
	}
}

func TestFieldDef_Defaults(t *testing.T) {
	r := BuildDefaultRegistry()

	// Check that non-required fields have defaults where specified
	def, _ := r.Get("ssh")
	for _, f := range def.Fields {
		if f.Name == "port" {
			if f.Default != 22 {
				t.Errorf("expected SSH port default to be 22, got %v", f.Default)
			}
		}
	}

	def, _ = r.Get("socks5")
	for _, f := range def.Fields {
		if f.Name == "udp" {
			if f.Default != true {
				t.Errorf("expected SOCKS5 udp default to be true, got %v", f.Default)
			}
		}
	}
}

func TestFieldDef_EnumValues(t *testing.T) {
	r := BuildDefaultRegistry()
	def, _ := r.Get("ss")

	var cipherField *FieldDef
	for i := range def.Fields {
		if def.Fields[i].Name == "cipher" {
			cipherField = &def.Fields[i]
			break
		}
	}
	if cipherField == nil {
		t.Fatal("expected cipher field in ss protocol")
	}
	if len(cipherField.EnumValues) != 7 {
		t.Errorf("expected 7 cipher enum values, got %d", len(cipherField.EnumValues))
	}
	if cipherField.Required != true {
		t.Error("expected cipher field to be required")
	}
}

func TestGet_EachProtocolHasRequiredFields(t *testing.T) {
	r := BuildDefaultRegistry()
	all := r.All()

	for _, pdef := range all {
		hasServer := false
		hasPort := false
		for _, f := range pdef.Fields {
			if f.Name == "server" && f.Required {
				hasServer = true
			}
			if f.Name == "port" && f.Required {
				hasPort = true
			}
		}
		if !hasServer {
			t.Errorf("protocol %s missing required server field", pdef.Type)
		}
		if !hasPort {
			t.Errorf("protocol %s missing required port field", pdef.Type)
		}
	}
}
