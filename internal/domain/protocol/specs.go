package protocol

func BuildDefaultRegistry() *Registry {
	r := New()

	r.Register(ProtocolDef{
		Type: "ss", Name: "Shadowsocks", Description: "Shadowsocks encrypted proxy",
		Transport: []string{"tcp", "udp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "cipher", Type: "enum", Required: true, Description: "加密方式",
				EnumValues: []string{"aes-128-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305", "none"}},
			{Name: "password", Type: "string", Required: false, Description: "密码"},
			{Name: "plugin", Type: "string", Required: false, Description: "SIP003 插件"},
			{Name: "udp", Type: "bool", Required: false, Description: "启用 UDP", Default: true},
		},
	})
	r.Register(ProtocolDef{
		Type: "vmess", Name: "VMess", Description: "V2Ray VMess 协议",
		Transport: []string{"tcp", "ws", "grpc", "h2"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "uuid", Type: "string", Required: true, Description: "用户 UUID"},
			{Name: "alterId", Type: "int", Required: false, Description: "额外 ID", Default: 0},
			{Name: "cipher", Type: "enum", Required: false, Description: "加密方式",
				EnumValues: []string{"auto", "aes-128-gcm", "chacha20-poly1305", "none"}},
			{Name: "network", Type: "enum", Required: false, Description: "传输方式",
				EnumValues: []string{"tcp", "ws", "grpc", "h2"}},
			{Name: "tls", Type: "bool", Required: false, Description: "启用 TLS", Default: false},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
		},
	})
	r.Register(ProtocolDef{
		Type: "trojan", Name: "Trojan", Description: "Trojan 代理协议",
		Transport: []string{"tcp", "ws", "grpc"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "password", Type: "string", Required: true, Description: "密码"},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
			{Name: "network", Type: "enum", Required: false, Description: "传输方式",
				EnumValues: []string{"tcp", "ws", "grpc"}},
			{Name: "alpn", Type: "[]string", Required: false, Description: "ALPN 协议"},
		},
	})
	r.Register(ProtocolDef{
		Type: "hysteria2", Name: "Hysteria2", Description: "Hysteria2 QUIC 协议",
		Transport: []string{"quic"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "password", Type: "string", Required: true, Description: "密码"},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
			{Name: "insecure", Type: "bool", Required: false, Description: "跳过证书验证", Default: false},
			{Name: "up", Type: "string", Required: false, Description: "上行带宽"},
			{Name: "down", Type: "string", Required: false, Description: "下行带宽"},
		},
	})
	r.Register(ProtocolDef{
		Type: "vless", Name: "VLESS", Description: "VLESS XTLS / Reality 协议",
		Transport: []string{"tcp", "ws", "grpc", "reality"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "uuid", Type: "string", Required: true, Description: "用户 UUID"},
			{Name: "flow", Type: "string", Required: false, Description: "XTLS 流控"},
			{Name: "network", Type: "enum", Required: false, Description: "传输方式",
				EnumValues: []string{"tcp", "ws", "grpc"}},
			{Name: "security", Type: "enum", Required: false, Description: "安全模式",
				EnumValues: []string{"none", "tls", "reality"}},
			{Name: "publicKey", Type: "string", Required: false, Description: "Reality 公钥"},
			{Name: "shortId", Type: "string", Required: false, Description: "Reality Short ID"},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
		},
	})
	r.Register(ProtocolDef{
		Type: "socks5", Name: "SOCKS5", Description: "SOCKS5 代理",
		Transport: []string{"tcp", "udp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "username", Type: "string", Required: false, Description: "用户名"},
			{Name: "password", Type: "string", Required: false, Description: "密码"},
			{Name: "udp", Type: "bool", Required: false, Description: "启用 UDP", Default: true},
		},
	})
	r.Register(ProtocolDef{
		Type: "http", Name: "HTTP/HTTPS", Description: "HTTP 代理",
		Transport: []string{"tcp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "username", Type: "string", Required: false, Description: "用户名"},
			{Name: "password", Type: "string", Required: false, Description: "密码"},
			{Name: "tls", Type: "bool", Required: false, Description: "启用 HTTPS", Default: false},
		},
	})
	r.Register(ProtocolDef{
		Type: "hysteria", Name: "Hysteria", Description: "Hysteria QUIC 协议",
		Transport: []string{"quic"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "auth", Type: "string", Required: true, Description: "认证字符串"},
			{Name: "up", Type: "string", Required: false, Description: "上行带宽"},
			{Name: "down", Type: "string", Required: false, Description: "下行带宽"},
			{Name: "obfs", Type: "string", Required: false, Description: "混淆密码"},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
		},
	})
	r.Register(ProtocolDef{
		Type: "tuic", Name: "TUIC", Description: "TUIC QUIC 协议",
		Transport: []string{"quic"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "uuid", Type: "string", Required: true, Description: "用户 UUID"},
			{Name: "password", Type: "string", Required: true, Description: "密码"},
			{Name: "congestion_control", Type: "enum", Required: false, Description: "拥塞控制",
				EnumValues: []string{"bbr", "new_reno", "cubic"}},
			{Name: "sni", Type: "string", Required: false, Description: "TLS SNI"},
		},
	})
	r.Register(ProtocolDef{
		Type: "wireguard", Name: "WireGuard", Description: "WireGuard 隧道",
		Transport: []string{"udp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "privateKey", Type: "string", Required: true, Description: "私钥"},
			{Name: "publicKey", Type: "string", Required: true, Description: "公钥"},
			{Name: "addresses", Type: "[]string", Required: false, Description: "本地地址"},
			{Name: "mtu", Type: "int", Required: false, Description: "MTU", Default: 1420},
		},
	})
	r.Register(ProtocolDef{
		Type: "snell", Name: "Snell", Description: "Snell 代理协议",
		Transport: []string{"tcp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "psk", Type: "string", Required: true, Description: "预共享密钥"},
			{Name: "version", Type: "int", Required: false, Description: "协议版本", Default: 4},
			{Name: "obfs", Type: "string", Required: false, Description: "混淆方式"},
		},
	})
	r.Register(ProtocolDef{
		Type: "ssr", Name: "ShadowsocksR", Description: "ShadowsocksR 协议",
		Transport: []string{"tcp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口"},
			{Name: "cipher", Type: "string", Required: true, Description: "加密方式"},
			{Name: "password", Type: "string", Required: true, Description: "密码"},
			{Name: "protocol", Type: "string", Required: false, Description: "混淆协议"},
			{Name: "obfs", Type: "string", Required: false, Description: "混淆方式"},
		},
	})
	r.Register(ProtocolDef{
		Type: "ssh", Name: "SSH", Description: "SSH 隧道代理",
		Transport: []string{"tcp"},
		Fields: []FieldDef{
			{Name: "server", Type: "string", Required: true, Description: "服务器地址"},
			{Name: "port", Type: "int", Required: true, Description: "服务器端口", Default: 22},
			{Name: "username", Type: "string", Required: true, Description: "用户名"},
			{Name: "password", Type: "string", Required: false, Description: "密码"},
			{Name: "privateKey", Type: "string", Required: false, Description: "私钥路径"},
		},
	})

	return r
}
