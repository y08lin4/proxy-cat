package profile

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestParseSubscriptionClashYAML(t *testing.T) {
	input := []byte(`
mixed-port: 7890
allow-lan: true
external-controller: 127.0.0.1:19090
secret: local-secret
proxies:
  - name: HK 01
    type: ss
    server: hk.example.com
    port: 8388
    cipher: aes-128-gcm
    password: secret
  - { name: "JP 01", type: vmess, server: jp.example.com, port: 443, uuid: abc, alterId: 0, tls: true }
proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - HK 01
      - JP 01
rules:
  - DOMAIN-SUFFIX,google.com,PROXY
  - MATCH,PROXY
`)
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	prof, err := ParseSubscription(input, ParseOptions{
		ProfileName:      "Main",
		SubscriptionName: "Sub",
		SubscriptionURL:  "https://example.com/sub",
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ParseSubscription() error = %v", err)
	}
	if prof.Name != "Main" {
		t.Fatalf("profile name = %q", prof.Name)
	}
	if len(prof.Proxies) != 2 {
		t.Fatalf("proxy count = %d", len(prof.Proxies))
	}
	if prof.Proxies[0].Name != "HK 01" || prof.Proxies[0].Port != 8388 {
		t.Fatalf("first proxy = %+v", prof.Proxies[0])
	}
	if prof.Proxies[0].RawOptions["cipher"] != "aes-128-gcm" {
		t.Fatalf("raw cipher was not preserved: %#v", prof.Proxies[0].RawOptions)
	}
	if prof.Proxies[1].RawOptions["tls"] != true {
		t.Fatalf("inline bool was not parsed: %#v", prof.Proxies[1].RawOptions)
	}
	if !prof.Settings.AllowLAN || prof.Settings.ExternalController != "127.0.0.1:19090" || prof.Settings.Secret != "local-secret" {
		t.Fatalf("settings = %+v", prof.Settings)
	}
	if len(prof.ProxyGroups) != 1 || prof.ProxyGroups[0].Name != "PROXY" || len(prof.ProxyGroups[0].Proxies) != 2 {
		t.Fatalf("parsed groups = %+v", prof.ProxyGroups)
	}
	if len(prof.Rules) != 2 || prof.Rules[0].Type != "DOMAIN-SUFFIX" || prof.Rules[1].Type != "MATCH" {
		t.Fatalf("parsed rules = %+v", prof.Rules)
	}
	if got := prof.Subscriptions[0].UpdatedAt; !got.Equal(now) {
		t.Fatalf("subscription updated_at = %v", got)
	}
}

func TestParseSubscriptionBase64(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(`
proxies:
  - { name: US, type: ss, server: us.example.com, port: 8388 }
`))
	prof, err := ParseSubscription([]byte(encoded), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseSubscription(base64) error = %v", err)
	}
	if len(prof.Proxies) != 1 || prof.Proxies[0].Name != "US" {
		t.Fatalf("proxies = %+v", prof.Proxies)
	}
}

func TestParseSubscriptionAddsDefaultGroupsAndRules(t *testing.T) {
	prof, err := ParseSubscription([]byte(`
proxies:
  - { name: HK, type: ss, server: hk.example.com, port: 8388 }
`), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseSubscription() error = %v", err)
	}
	if len(prof.ProxyGroups) != 3 || prof.ProxyGroups[0].Name != "PROXY" || prof.ProxyGroups[1].Name != "AUTO" || prof.ProxyGroups[2].Name != "AUTO-STABLE" {
		t.Fatalf("default groups = %+v", prof.ProxyGroups)
	}
	if prof.ProxyGroups[2].Type != "auto-stable" {
		t.Fatalf("auto-stable default group = %+v", prof.ProxyGroups[2])
	}
	if len(prof.Rules) != 2 || prof.Rules[0].Type != "GEOIP" || prof.Rules[1].Type != "MATCH" {
		t.Fatalf("default rules = %+v", prof.Rules)
	}
}

func TestParseSubscriptionAllowsAutoStableGroup(t *testing.T) {
	prof, err := ParseSubscription([]byte(`
proxies:
  - { name: HK, type: ss, server: hk.example.com, port: 8388 }
proxy-groups:
  - name: AUTO-STABLE
    type: auto-stable
    proxies:
      - HK
`), ParseOptions{})
	if err != nil {
		t.Fatalf("ParseSubscription() error = %v", err)
	}
	if len(prof.ProxyGroups) != 1 {
		t.Fatalf("group count = %d", len(prof.ProxyGroups))
	}
	group := prof.ProxyGroups[0]
	if group.Name != "AUTO-STABLE" || group.Type != "auto-stable" || len(group.Proxies) != 1 || group.Proxies[0] != "HK" {
		t.Fatalf("auto-stable group = %+v", group)
	}
}

func TestParseSubscriptionRejectsUnsupportedInput(t *testing.T) {
	_, err := ParseSubscription([]byte("proxies:\n  - name: Bad\n    type: ss\n"), ParseOptions{})
	if err == nil {
		t.Fatal("ParseSubscription() expected an error")
	}
}
