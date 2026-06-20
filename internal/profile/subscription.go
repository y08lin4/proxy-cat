package profile

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ParseOptions struct {
	ProfileID        string
	ProfileName      string
	SubscriptionID   string
	SubscriptionName string
	SubscriptionURL  string
	Now              time.Time
}

func ParseSubscription(data []byte, opts ParseOptions) (Profile, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return Profile{}, errors.New("subscription is empty")
	}

	decoded := decodeBase64IfLikely(data)
	lines := splitMeaningfulLines(decoded)
	proxies, err := parseClashProxies(lines)
	if err != nil {
		return Profile{}, err
	}
	if len(proxies) == 0 && !bytes.Equal(decoded, data) {
		lines = splitMeaningfulLines(data)
		proxies, err = parseClashProxies(lines)
		if err != nil {
			return Profile{}, err
		}
	}
	if len(proxies) == 0 {
		return Profile{}, errors.New("subscription contains no supported proxies")
	}
	settings := parseClashSettings(lines)
	groups, err := parseClashProxyGroups(lines)
	if err != nil {
		return Profile{}, err
	}
	rules := parseClashRules(lines)

	name := strings.TrimSpace(opts.ProfileName)
	if name == "" {
		name = strings.TrimSpace(opts.SubscriptionName)
	}
	if name == "" {
		name = "Imported Subscription"
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	profile := Profile{
		ID:       strings.TrimSpace(opts.ProfileID),
		Name:     name,
		Proxies:  proxies,
		Settings: settings,
	}
	if opts.SubscriptionURL != "" || opts.SubscriptionName != "" || opts.SubscriptionID != "" {
		profile.Subscriptions = []Subscription{{
			ID:        strings.TrimSpace(opts.SubscriptionID),
			Name:      strings.TrimSpace(opts.SubscriptionName),
			URL:       strings.TrimSpace(opts.SubscriptionURL),
			UpdatedAt: now,
		}}
	}
	if len(groups) == 0 {
		groups = DefaultGroups(profile.ProxyNames())
	}
	if len(rules) == 0 {
		rules = DefaultRules("PROXY")
	}
	profile.ProxyGroups = groups
	profile.Rules = rules
	return profile, nil
}

func decodeBase64IfLikely(data []byte) []byte {
	text := strings.TrimSpace(string(data))
	if strings.Contains(text, "\n") || strings.Contains(text, "proxies:") {
		return data
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err == nil && len(bytes.TrimSpace(decoded)) > 0 {
		return bytes.TrimSpace(decoded)
	}
	decoded, err = base64.RawStdEncoding.DecodeString(text)
	if err == nil && len(bytes.TrimSpace(decoded)) > 0 {
		return bytes.TrimSpace(decoded)
	}
	return data
}

func splitMeaningfulLines(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var lines []string
	for scanner.Scan() {
		line := stripComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func parseClashProxies(lines []string) ([]ProxyNode, error) {
	var proxies []ProxyNode
	inProxies := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !inProxies {
			if trimmed == "proxies:" {
				inProxies = true
			}
			continue
		}
		if isTopLevelKey(line) && trimmed != "proxies:" {
			break
		}
		if !strings.HasPrefix(strings.TrimLeft(line, " "), "-") {
			continue
		}

		itemLines := []string{strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " "), "-"))}
		for i+1 < len(lines) {
			next := lines[i+1]
			if strings.HasPrefix(strings.TrimLeft(next, " "), "-") || isTopLevelKey(next) {
				break
			}
			itemLines = append(itemLines, strings.TrimSpace(next))
			i++
		}
		fields, err := parseMapItem(itemLines)
		if err != nil {
			return nil, err
		}
		if len(fields) == 0 {
			continue
		}
		proxy, err := proxyFromFields(fields)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

func parseClashSettings(lines []string) Settings {
	settings := DefaultSettings()
	haveMixedPort := false
	for _, line := range lines {
		if !isTopLevelKey(line) {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}
		scalar := parseScalar(value)
		switch strings.TrimSpace(key) {
		case "mixed-port":
			if port, ok := scalarAsInt(scalar); ok {
				settings.MixedPort = port
				haveMixedPort = true
			}
		case "port":
			if !haveMixedPort {
				if port, ok := scalarAsInt(scalar); ok {
					settings.MixedPort = port
				}
			}
		case "allow-lan":
			if allowLAN, ok := scalar.(bool); ok {
				settings.AllowLAN = allowLAN
			}
		case "external-controller":
			settings.ExternalController = strings.TrimSpace(fmt.Sprint(scalar))
		case "secret":
			settings.Secret = strings.TrimSpace(fmt.Sprint(scalar))
		}
	}
	if settings.MixedPort <= 0 {
		settings.MixedPort = 7890
	}
	if settings.ExternalController == "" {
		settings.ExternalController = "127.0.0.1:9090"
	}
	return settings
}

func parseClashProxyGroups(lines []string) ([]ProxyGroup, error) {
	var groups []ProxyGroup
	inGroups := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !inGroups {
			if trimmed == "proxy-groups:" {
				inGroups = true
			}
			continue
		}
		if isTopLevelKey(line) && trimmed != "proxy-groups:" {
			break
		}
		if !strings.HasPrefix(strings.TrimLeft(line, " "), "-") {
			continue
		}

		itemLines := []string{strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " "), "-"))}
		for i+1 < len(lines) {
			next := lines[i+1]
			if isTopLevelListItem(next) || isTopLevelKey(next) {
				break
			}
			itemLines = append(itemLines, strings.TrimSpace(next))
			i++
		}
		group, err := proxyGroupFromItemLines(itemLines)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(group.Name) != "" {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func proxyGroupFromItemLines(lines []string) (ProxyGroup, error) {
	fields := make(map[string]any)
	var listKey string
	for _, line := range lines {
		if strings.HasPrefix(line, "-") && listKey != "" {
			fields[listKey] = appendStringField(fields[listKey], strings.TrimSpace(strings.TrimPrefix(line, "-")))
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return ProxyGroup{}, fmt.Errorf("unsupported proxy-group line %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if value == "" {
			listKey = key
			continue
		}
		fields[key] = parseScalar(value)
		listKey = ""
	}
	group := ProxyGroup{
		Name:          stringField(fields, "name"),
		Type:          stringField(fields, "type"),
		SelectedProxy: stringField(fields, "selected"),
	}
	group.Proxies = scalarAsStrings(fields["proxies"])
	return group, nil
}

func parseClashRules(lines []string) []Rule {
	var rules []Rule
	inRules := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inRules {
			if trimmed == "rules:" {
				inRules = true
			}
			continue
		}
		if isTopLevelKey(line) && trimmed != "rules:" {
			break
		}
		if !strings.HasPrefix(strings.TrimLeft(line, " "), "-") {
			continue
		}
		ruleLine := strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(line, " "), "-"))
		ruleLine = unquote(ruleLine)
		parts := splitCSVRespectingQuotes(ruleLine)
		if len(parts) < 2 {
			continue
		}
		rule := Rule{Type: strings.TrimSpace(unquote(parts[0]))}
		if strings.EqualFold(rule.Type, "MATCH") {
			rule.TargetGroup = strings.TrimSpace(unquote(parts[len(parts)-1]))
		} else {
			rule.Value = strings.TrimSpace(unquote(parts[1]))
			rule.TargetGroup = strings.TrimSpace(unquote(parts[len(parts)-1]))
		}
		rules = append(rules, rule)
	}
	return rules
}

func parseMapItem(lines []string) (map[string]any, error) {
	fields := make(map[string]any)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			inline, err := parseInlineMap(line)
			if err != nil {
				return nil, err
			}
			for key, value := range inline {
				fields[key] = value
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("unsupported proxy line %q", line)
		}
		fields[strings.TrimSpace(key)] = parseScalar(strings.TrimSpace(value))
	}
	return fields, nil
}

func parseInlineMap(line string) (map[string]any, error) {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(strings.TrimSuffix(line, "}"), "{")
	parts := splitCSVRespectingQuotes(line)
	fields := make(map[string]any, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("unsupported inline field %q", strings.TrimSpace(part))
		}
		fields[strings.TrimSpace(unquote(key))] = parseScalar(strings.TrimSpace(value))
	}
	return fields, nil
}

func proxyFromFields(fields map[string]any) (ProxyNode, error) {
	name := stringField(fields, "name")
	typ := stringField(fields, "type")
	server := stringField(fields, "server")
	port, ok := intField(fields, "port")
	if !ok {
		return ProxyNode{}, fmt.Errorf("proxy %q has invalid port", name)
	}
	raw := make(map[string]any, len(fields))
	for key, value := range fields {
		raw[key] = value
	}
	proxy := ProxyNode{
		ID:         stableID(name),
		Name:       name,
		Type:       typ,
		Server:     server,
		Port:       port,
		RawOptions: raw,
	}
	if err := proxy.Validate(); err != nil {
		return ProxyNode{}, err
	}
	return proxy, nil
}

func parseScalar(value string) any {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(value, "]"), "["))
		if inner == "" {
			return []any{}
		}
		parts := splitCSVRespectingQuotes(inner)
		items := make([]any, 0, len(parts))
		for _, part := range parts {
			items = append(items, parseScalar(part))
		}
		return items
	}
	unquoted := unquote(value)
	if i, err := strconv.Atoi(unquoted); err == nil {
		return i
	}
	if b, err := strconv.ParseBool(strings.ToLower(unquoted)); err == nil {
		return b
	}
	return unquoted
}

func splitCSVRespectingQuotes(s string) []string {
	var parts []string
	var b strings.Builder
	var quote rune
	escaped := false
	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			b.WriteRune(r)
			escaped = true
			continue
		}
		if quote != 0 {
			b.WriteRune(r)
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			b.WriteRune(r)
			continue
		}
		if r == ',' {
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	if strings.TrimSpace(b.String()) != "" {
		parts = append(parts, strings.TrimSpace(b.String()))
	}
	return parts
}

func stripComment(line string) string {
	var quote rune
	for i, r := range line {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '#' {
			return line[:i]
		}
	}
	return line
}

func isTopLevelKey(line string) bool {
	if line == "" || line[0] == ' ' || line[0] == '\t' {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "-") {
		return false
	}
	key, _, ok := strings.Cut(trimmed, ":")
	return ok && key != ""
}

func isTopLevelListItem(line string) bool {
	return countLeadingSpaces(line) <= 2 && strings.HasPrefix(strings.TrimLeft(line, " "), "-")
}

func countLeadingSpaces(line string) int {
	count := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func stringField(fields map[string]any, key string) string {
	value, ok := fields[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func intField(fields map[string]any, key string) (int, bool) {
	value, ok := fields[key]
	if !ok {
		return 0, false
	}
	return scalarAsInt(value)
}

func scalarAsInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	default:
		return 0, false
	}
}

func scalarAsStrings(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, strings.TrimSpace(unquote(fmt.Sprint(item))))
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(unquote(v))}
	default:
		return nil
	}
}

func appendStringField(value any, item string) []any {
	items, _ := value.([]any)
	return append(items, parseScalar(item))
}

func stableID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "proxy"
	}
	return b.String()
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
