package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// ParseURIList parses a list of proxy URIs (one per line: ss://, vmess://, trojan://, etc.)
func ParseURIList(data []byte) ([]proxy.ProxyNode, error) {
	text := strings.TrimSpace(string(data))
	lines := strings.Split(text, "\n")
	var proxies []proxy.ProxyNode
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		node, err := ParseURI(line)
		if err != nil {
			continue // skip unparseable URIs
		}
		proxies = append(proxies, node)
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("no supported proxies found in URI list")
	}
	return proxies, nil
}

// ParseURI parses a single proxy URI and returns a ProxyNode.
func ParseURI(uri string) (proxy.ProxyNode, error) {
	switch {
	case strings.HasPrefix(uri, "ssr://"):
		return parseSSRURI(uri)
	case strings.HasPrefix(uri, "ss://"):
		return parseSSURI(uri)
	case strings.HasPrefix(uri, "vmess://"):
		return parseVMessURI(uri)
	case strings.HasPrefix(uri, "trojan://"):
		return parseTrojanURI(uri)
	case strings.HasPrefix(uri, "vless://"):
		return parseVLESSURI(uri)
	case strings.HasPrefix(uri, "hysteria2://"), strings.HasPrefix(uri, "hy2://"):
		return parseHysteria2URI(uri)
	case strings.HasPrefix(uri, "hysteria://"):
		return parseHysteriaURI(uri)
	case strings.HasPrefix(uri, "tuic://"):
		return parseTUICURI(uri)
	default:
		return proxy.ProxyNode{}, fmt.Errorf("unsupported URI scheme: %s", uri)
	}
}

// ss://base64(method:password)@host:port  or  ss://method:password@host:port
func parseSSURI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}

	var method, password string
	userInfo := u.User.String()
	if strings.Contains(userInfo, ":") {
		parts := strings.SplitN(userInfo, ":", 2)
		method, password = parts[0], parts[1]
	} else {
		decoded, err := base64.RawURLEncoding.DecodeString(userInfo)
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(userInfo)
			if err != nil {
				return proxy.ProxyNode{}, err
			}
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return proxy.ProxyNode{}, fmt.Errorf("invalid ss URI userinfo")
		}
		method, password = parts[0], parts[1]
	}

	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}

	return proxy.ProxyNode{
		ID:     stableID(name),
		Name:   name,
		Type:   "ss",
		Server: u.Hostname(),
		Port:   port,
		RawOptions: map[string]any{
			"cipher":   method,
			"password": password,
		},
	}, nil
}

func parseTrojanURI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}

	raw := map[string]any{"password": u.User.String()}
	if sni := u.Query().Get("sni"); sni != "" {
		raw["sni"] = sni
	}

	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "trojan",
		Server: u.Hostname(), Port: port, RawOptions: raw,
	}, nil
}

func parseVMessURI(uri string) (proxy.ProxyNode, error) {
	encoded := strings.TrimPrefix(uri, "vmess://")
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return proxy.ProxyNode{}, err
		}
	}
	var cfg map[string]any
	if err := json.Unmarshal(decoded, &cfg); err != nil {
		return proxy.ProxyNode{}, err
	}
	name, _ := cfg["ps"].(string)
	if name == "" {
		name, _ = cfg["add"].(string)
	}
	port := 443
	if p, ok := cfg["port"]; ok {
		switch v := p.(type) {
		case float64:
			port = int(v)
		case string:
			port, _ = strconv.Atoi(v)
		}
	}
	raw := make(map[string]any)
	for k, v := range cfg {
		if k == "ps" || k == "add" || k == "port" {
			continue
		}
		raw[k] = v
	}
	server, _ := cfg["add"].(string)
	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "vmess",
		Server: server, Port: port, RawOptions: raw,
	}, nil
}

func parseVLESSURI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	raw := map[string]any{"uuid": u.User.String()}
	for k, v := range u.Query() {
		if len(v) > 0 {
			raw[k] = v[0]
		}
	}
	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "vless",
		Server: u.Hostname(), Port: port, RawOptions: raw,
	}, nil
}

func parseHysteria2URI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	raw := map[string]any{"password": u.User.String()}
	if sni := u.Query().Get("sni"); sni != "" {
		raw["sni"] = sni
	}
	if insecure := u.Query().Get("insecure"); insecure == "1" {
		raw["insecure"] = true
	}
	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "hysteria2",
		Server: u.Hostname(), Port: port, RawOptions: raw,
	}, nil
}

func parseHysteriaURI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	raw := map[string]any{"auth": u.User.String()}
	for k, v := range u.Query() {
		if len(v) > 0 {
			raw[k] = v[0]
		}
	}
	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "hysteria",
		Server: u.Hostname(), Port: port, RawOptions: raw,
	}, nil
}

func parseTUICURI(uri string) (proxy.ProxyNode, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return proxy.ProxyNode{}, err
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	userInfo := u.User.String()
	parts := strings.SplitN(userInfo, ":", 2)
	raw := make(map[string]any)
	if len(parts) == 2 {
		raw["uuid"] = parts[0]
		raw["password"] = parts[1]
	}
	for k, v := range u.Query() {
		if len(v) > 0 {
			raw[k] = v[0]
		}
	}
	return proxy.ProxyNode{
		ID: stableID(name), Name: name, Type: "tuic",
		Server: u.Hostname(), Port: port, RawOptions: raw,
	}, nil
}

func parseSSRURI(uri string) (proxy.ProxyNode, error) {
	return proxy.ProxyNode{}, fmt.Errorf("ssr:// URI parsing not yet implemented")
}
