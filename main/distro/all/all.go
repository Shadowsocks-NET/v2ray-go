package all

import (
	// The following are necessary as they register handlers in their init functions.

	// Mandatory features. Can't remove unless there are replacements.
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/dispatcher"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/proxyman/inbound"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/proxyman/outbound"

	// Default commander and all its services. This is an optional feature.
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/commander"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/log/command"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/proxyman/command"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/stats/command"

	// Developer preview services
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/observatory/command"

	// Other optional features.
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/dns"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/dns/fakedns"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/log"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/policy"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/reverse"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/router"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/stats"

	// Fix dependency cycle caused by core import in internet package
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/tagged/taggedimpl"

	// Developer preview features
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/app/observatory"

	// Inbound and outbound proxies.
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/blackhole"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/dns"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/dokodemo"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/freedom"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/http"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/shadowsocks"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/socks"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/trojan"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vless/inbound"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vless/outbound"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/inbound"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/outbound"

	// Transports
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/domainsocket"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/grpc"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/http"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/kcp"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/quic"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/tcp"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/tls"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/udp"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/websocket"

	// Transport headers
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/http"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/noop"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/srtp"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/tls"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/utp"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/wechat"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/headers/wireguard"

	// Geo loaders
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/geodata/memconservative"
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/geodata/standard"

	// JSON config support. Choose only one from the two below.
	// The following line loads JSON from v2ctl
	// _ "github.com/Shadowsocks-NET/v2ray-go/v4/main/json"
	// The following line loads JSON internally
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/main/jsonem"

	// Load config from file or http(s)
	_ "github.com/Shadowsocks-NET/v2ray-go/v4/main/confloader/external"
)
