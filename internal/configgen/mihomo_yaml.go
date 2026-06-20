package configgen

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/y08lin4/proxy-cat/internal/profile"
)

type Options struct {
	MixedPort          int
	AllowLAN           *bool
	ExternalController string
	Secret             string
}

func GenerateMihomoYAML(p profile.Profile, opts Options) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	settings := profile.DefaultSettings()
	if p.Settings.MixedPort > 0 {
		settings.MixedPort = p.Settings.MixedPort
	}
	settings.AllowLAN = p.Settings.AllowLAN
	if p.Settings.ExternalController != "" {
		settings.ExternalController = p.Settings.ExternalController
	}
	if p.Settings.Secret != "" {
		settings.Secret = p.Settings.Secret
	}
	if opts.MixedPort > 0 {
		settings.MixedPort = opts.MixedPort
	}
	if opts.AllowLAN != nil {
		settings.AllowLAN = *opts.AllowLAN
	}
	if opts.ExternalController != "" {
		settings.ExternalController = opts.ExternalController
	}
	if opts.Secret != "" {
		settings.Secret = opts.Secret
	}

	groups := p.ProxyGroups
	if len(groups) == 0 {
		groups = profile.DefaultGroups(p.ProxyNames())
	}
	rules := p.Rules
	if len(rules) == 0 {
		rules = profile.DefaultRules("PROXY")
	}

	var b bytes.Buffer
	writeKV(&b, "mixed-port", settings.MixedPort)
	writeKV(&b, "allow-lan", settings.AllowLAN)
	writeKV(&b, "external-controller", settings.ExternalController)
	writeKV(&b, "secret", settings.Secret)
	b.WriteString("proxies:\n")
	for _, proxy := range p.Proxies {
		writeProxy(&b, proxy)
	}
	b.WriteString("proxy-groups:\n")
	for _, group := range groups {
		writeProxyGroup(&b, group)
	}
	b.WriteString("rules:\n")
	for _, rule := range rules {
		b.WriteString("  - ")
		b.WriteString(formatRule(rule))
		b.WriteByte('\n')
	}
	b.WriteString("dns:\n")
	b.WriteString("  enable: true\n")
	b.WriteString("  enhanced-mode: fake-ip\n")
	b.WriteString("  nameserver:\n")
	b.WriteString("    - 223.5.5.5\n")
	b.WriteString("    - 1.1.1.1\n")
	return b.Bytes(), nil
}

func writeProxy(b *bytes.Buffer, proxy profile.ProxyNode) {
	b.WriteString("  - name: ")
	b.WriteString(quoteString(proxy.Name))
	b.WriteByte('\n')
	fields := make(map[string]any, len(proxy.RawOptions)+4)
	for key, value := range proxy.RawOptions {
		fields[key] = value
	}
	fields["type"] = proxy.Type
	fields["server"] = proxy.Server
	fields["port"] = proxy.Port
	delete(fields, "name")

	for _, key := range orderedProxyKeys(fields) {
		b.WriteString("    ")
		b.WriteString(key)
		b.WriteString(": ")
		writeScalar(b, fields[key])
		b.WriteByte('\n')
	}
}

func writeProxyGroup(b *bytes.Buffer, group profile.ProxyGroup) {
	b.WriteString("  - name: ")
	b.WriteString(quoteString(group.Name))
	b.WriteByte('\n')
	b.WriteString("    type: ")
	groupType := strings.TrimSpace(group.Type)
	if strings.EqualFold(groupType, "auto-stable") {
		groupType = "select"
	}
	b.WriteString(quoteString(groupType))
	b.WriteByte('\n')
	if strings.EqualFold(strings.TrimSpace(group.Type), "url-test") {
		b.WriteString("    url: https://www.gstatic.com/generate_204\n")
		b.WriteString("    interval: 300\n")
	}
	b.WriteString("    proxies:\n")
	for _, name := range group.Proxies {
		b.WriteString("      - ")
		b.WriteString(quoteString(name))
		b.WriteByte('\n')
	}
}

func writeKV(b *bytes.Buffer, key string, value any) {
	b.WriteString(key)
	b.WriteString(": ")
	writeScalar(b, value)
	b.WriteByte('\n')
}

func writeScalar(b *bytes.Buffer, value any) {
	switch v := value.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(v))
	case int:
		b.WriteString(strconv.Itoa(v))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case []any:
		b.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				b.WriteString(", ")
			}
			writeScalar(b, item)
		}
		b.WriteByte(']')
	case []string:
		b.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteString(item))
		}
		b.WriteByte(']')
	default:
		b.WriteString(quoteString(fmt.Sprint(v)))
	}
}

func formatRule(rule profile.Rule) string {
	typ := strings.TrimSpace(rule.Type)
	value := strings.TrimSpace(rule.Value)
	target := strings.TrimSpace(rule.TargetGroup)
	if strings.EqualFold(typ, "MATCH") {
		return "MATCH," + target
	}
	if value == "" {
		return typ + "," + target
	}
	return typ + "," + value + "," + target
}

func orderedProxyKeys(fields map[string]any) []string {
	priority := []string{"type", "server", "port", "cipher", "password", "uuid", "alterId", "tls", "network"}
	seen := make(map[string]struct{}, len(fields))
	keys := make([]string, 0, len(fields))
	for _, key := range priority {
		if _, ok := fields[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}
	var rest []string
	for key := range fields {
		if _, ok := seen[key]; !ok {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func quoteString(s string) string {
	if s == "" {
		return `""`
	}
	if isPlainYAMLString(s) {
		return s
	}
	return strconv.Quote(s)
}

func isPlainYAMLString(s string) bool {
	switch strings.ToLower(s) {
	case "true", "false", "null", "~":
		return false
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == '/' || r == ':':
		default:
			return false
		}
	}
	return !strings.Contains(s, ": ")
}
