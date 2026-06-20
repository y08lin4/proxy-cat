package configgen

import (
	"strings"
	"testing"

	"github.com/y08lin4/proxy-cat/internal/profile"
)

func TestGenerateMihomoYAML(t *testing.T) {
	p := profile.Profile{
		Name: "Main",
		Proxies: []profile.ProxyNode{
			{
				Name:   "HK 01",
				Type:   "ss",
				Server: "hk.example.com",
				Port:   8388,
				RawOptions: map[string]any{
					"name":     "HK 01",
					"type":     "ss",
					"server":   "hk.example.com",
					"port":     8388,
					"cipher":   "aes-128-gcm",
					"password": "secret",
				},
			},
			{
				Name:   "US",
				Type:   "vmess",
				Server: "us.example.com",
				Port:   443,
				RawOptions: map[string]any{
					"uuid": "abc",
					"tls":  true,
				},
			},
		},
		Settings: profile.Settings{Secret: "local-secret"},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}
	yaml := string(got)
	assertContains(t, yaml, "mixed-port: 7890\n")
	assertContains(t, yaml, "allow-lan: false\n")
	assertContains(t, yaml, "external-controller: 127.0.0.1:9090\n")
	assertContains(t, yaml, "secret: local-secret\n")
	assertContains(t, yaml, "  - name: \"HK 01\"\n")
	assertContains(t, yaml, "    cipher: aes-128-gcm\n")
	assertContains(t, yaml, "    password: secret\n")
	assertContains(t, yaml, "  - name: US\n")
	assertContains(t, yaml, "    uuid: abc\n")
	assertContains(t, yaml, "proxy-groups:\n")
	assertContains(t, yaml, "  - name: PROXY\n")
	assertContains(t, yaml, "      - AUTO-STABLE\n")
	assertContains(t, yaml, "      - AUTO\n")
	assertContains(t, yaml, "      - DIRECT\n")
	assertContains(t, yaml, "  - name: AUTO\n")
	assertContains(t, yaml, "    type: url-test\n")
	assertContains(t, yaml, "  - name: AUTO-STABLE\n")
	assertContains(t, yaml, "    type: select\n")
	assertContains(t, yaml, "rules:\n")
	assertContains(t, yaml, "  - GEOIP,CN,DIRECT\n")
	assertContains(t, yaml, "  - MATCH,PROXY\n")
	assertContains(t, yaml, "dns:\n")
}

func TestGenerateMihomoYAMLMapsAutoStableToSelect(t *testing.T) {
	p := profile.Profile{
		Name: "Main",
		Proxies: []profile.ProxyNode{{
			Name:   "HK",
			Type:   "ss",
			Server: "hk.example.com",
			Port:   8388,
		}},
		ProxyGroups: []profile.ProxyGroup{{
			Name:    "AUTO-STABLE",
			Type:    "auto-stable",
			Proxies: []string{"HK"},
		}},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}
	yaml := string(got)
	assertContains(t, yaml, "  - name: AUTO-STABLE\n")
	assertContains(t, yaml, "    type: select\n")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected YAML to contain %q\nfull YAML:\n%s", needle, haystack)
	}
}
