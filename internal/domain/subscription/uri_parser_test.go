package subscription

import (
	"encoding/base64"
	"testing"
)

func TestParseSSURI(t *testing.T) {
	// SIP002 format: ss://method:password@host:port#name
	node, err := ParseURI("ss://aes-256-gcm:mypassword@hk.example.com:8388#HK-Server")
	if err != nil {
		t.Fatalf("ParseURI(ss://) error = %v", err)
	}
	if node.Name != "HK-Server" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "ss" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "hk.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 8388 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["cipher"] != "aes-256-gcm" {
		t.Fatalf("cipher = %v", node.RawOptions["cipher"])
	}
	if node.RawOptions["password"] != "mypassword" {
		t.Fatalf("password = %v", node.RawOptions["password"])
	}
}

func TestParseTrojanURI(t *testing.T) {
	node, err := ParseURI("trojan://password123@us.example.com:443#US-Trojan")
	if err != nil {
		t.Fatalf("ParseURI(trojan://) error = %v", err)
	}
	if node.Name != "US-Trojan" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "trojan" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "us.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 443 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["password"] != "password123" {
		t.Fatalf("password = %v", node.RawOptions["password"])
	}
}

func TestParseTrojanURIWithSNI(t *testing.T) {
	node, err := ParseURI("trojan://password123@us.example.com:443?sni=cdn.example.com#US-Trojan")
	if err != nil {
		t.Fatalf("ParseURI(trojan://?sni) error = %v", err)
	}
	if node.RawOptions["sni"] != "cdn.example.com" {
		t.Fatalf("sni = %v", node.RawOptions["sni"])
	}
}

func TestParseVMessURI(t *testing.T) {
	cfg := `{"v":"2","ps":"JP-VMess","add":"jp.example.com","port":443,"id":"abc","aid":0,"net":"tcp","type":"none","tls":"tls"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(cfg))
	uri := "vmess://" + encoded

	node, err := ParseURI(uri)
	if err != nil {
		t.Fatalf("ParseURI(vmess://) error = %v", err)
	}
	if node.Name != "JP-VMess" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "vmess" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "jp.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 443 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["id"] != "abc" {
		t.Fatalf("id = %v", node.RawOptions["id"])
	}
}

func TestParseVLESSURI(t *testing.T) {
	node, err := ParseURI("vless://abc-def-ghi@jp.example.com:443?type=tcp&security=tls#JP-VLESS")
	if err != nil {
		t.Fatalf("ParseURI(vless://) error = %v", err)
	}
	if node.Name != "JP-VLESS" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "vless" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "jp.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 443 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["uuid"] != "abc-def-ghi" {
		t.Fatalf("uuid = %v", node.RawOptions["uuid"])
	}
	if node.RawOptions["type"] != "tcp" {
		t.Fatalf("type = %v", node.RawOptions["type"])
	}
}

func TestParseHysteria2URI(t *testing.T) {
	node, err := ParseURI("hysteria2://password123@sg.example.com:8443?sni=cdn.example.com#SG-Hy2")
	if err != nil {
		t.Fatalf("ParseURI(hysteria2://) error = %v", err)
	}
	if node.Name != "SG-Hy2" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "hysteria2" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "sg.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 8443 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["password"] != "password123" {
		t.Fatalf("password = %v", node.RawOptions["password"])
	}
	if node.RawOptions["sni"] != "cdn.example.com" {
		t.Fatalf("sni = %v", node.RawOptions["sni"])
	}
}

func TestParseHysteria2URIAltScheme(t *testing.T) {
	node, err := ParseURI("hy2://password123@sg.example.com:8443#SG-Hy2")
	if err != nil {
		t.Fatalf("ParseURI(hy2://) error = %v", err)
	}
	if node.Type != "hysteria2" {
		t.Fatalf("type = %q", node.Type)
	}
}

func TestParseHysteriaURI(t *testing.T) {
	node, err := ParseURI("hysteria://auth-string@jp.example.com:443?insecure=1&sni=jp.example.com#JP-Hysteria")
	if err != nil {
		t.Fatalf("ParseURI(hysteria://) error = %v", err)
	}
	if node.Type != "hysteria" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.RawOptions["auth"] != "auth-string" {
		t.Fatalf("auth = %v", node.RawOptions["auth"])
	}
}

func TestParseTUICURI(t *testing.T) {
	node, err := ParseURI("tuic://abc-def:password123@hk.example.com:8443?congestion_control=bbr&sni=hk.example.com#HK-TUIC")
	if err != nil {
		t.Fatalf("ParseURI(tuic://) error = %v", err)
	}
	if node.Name != "HK-TUIC" {
		t.Fatalf("name = %q", node.Name)
	}
	if node.Type != "tuic" {
		t.Fatalf("type = %q", node.Type)
	}
	if node.Server != "hk.example.com" {
		t.Fatalf("server = %q", node.Server)
	}
	if node.Port != 8443 {
		t.Fatalf("port = %d", node.Port)
	}
	if node.RawOptions["uuid"] != "abc-def" {
		t.Fatalf("uuid = %v", node.RawOptions["uuid"])
	}
	if node.RawOptions["password"] != "password123" {
		t.Fatalf("password = %v", node.RawOptions["password"])
	}
	if node.RawOptions["congestion_control"] != "bbr" {
		t.Fatalf("congestion_control = %v", node.RawOptions["congestion_control"])
	}
}

func TestParseURIUnsupportedScheme(t *testing.T) {
	_, err := ParseURI("unknown://example.com")
	if err == nil {
		t.Fatal("ParseURI(unknown://) expected an error")
	}
}

func TestParseSSRURINotImplemented(t *testing.T) {
	_, err := ParseURI("ssr://base64data")
	if err == nil {
		t.Fatal("ParseURI(ssr://) expected an error")
	}
}

func TestParseURIList(t *testing.T) {
	input := []byte(`ss://aes-256-gcm:mypassword@hk.example.com:8388#HK
vmess://` + base64.StdEncoding.EncodeToString([]byte(`{"v":"2","ps":"JP","add":"jp.example.com","port":443,"id":"abc","aid":0}`)) + `
trojan://password123@us.example.com:443#US
vless://abc-def@jp.example.com:443?security=tls#JP
hysteria2://password123@sg.example.com:8443#SG
`)

	proxies, err := ParseURIList(input)
	if err != nil {
		t.Fatalf("ParseURIList error = %v", err)
	}
	if len(proxies) != 5 {
		t.Fatalf("expected 5 proxies, got %d", len(proxies))
	}
	types := []string{"ss", "vmess", "trojan", "vless", "hysteria2"}
	for i, typ := range types {
		if proxies[i].Type != typ {
			t.Fatalf("proxy[%d].type = %q, want %q", i, proxies[i].Type, typ)
		}
	}
}

func TestParseURIListSkipsInvalid(t *testing.T) {
	input := []byte(`ss://aes-256-gcm:mypassword@hk.example.com:8388#HK
this is not a valid uri
trojan://password123@us.example.com:443#US
`)
	proxies, err := ParseURIList(input)
	if err != nil {
		t.Fatalf("ParseURIList error = %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(proxies))
	}
}

func TestParseSubscriptionURIList(t *testing.T) {
	input := []byte(`ss://aes-256-gcm:mypassword@hk.example.com:8388#HK
trojan://password123@us.example.com:443#US
vless://abc-def@jp.example.com:443?security=tls#JP
`)
	prof, err := ParseSubscription(input, ParseOptions{ProfileName: "URI Profile"})
	if err != nil {
		t.Fatalf("ParseSubscription(URI list) error = %v", err)
	}
	if prof.Name != "URI Profile" {
		t.Fatalf("profile name = %q", prof.Name)
	}
	if len(prof.Proxies) != 3 {
		t.Fatalf("expected 3 proxies, got %d", len(prof.Proxies))
	}
	if prof.Proxies[0].Type != "ss" {
		t.Fatalf("proxy[0].type = %q", prof.Proxies[0].Type)
	}
	if prof.Proxies[1].Type != "trojan" {
		t.Fatalf("proxy[1].type = %q", prof.Proxies[1].Type)
	}
	if prof.Proxies[2].Type != "vless" {
		t.Fatalf("proxy[2].type = %q", prof.Proxies[2].Type)
	}
	// Should have default groups and rules
	if len(prof.ProxyGroups) == 0 {
		t.Fatal("expected default proxy groups")
	}
	if len(prof.Rules) == 0 {
		t.Fatal("expected default rules")
	}
}
