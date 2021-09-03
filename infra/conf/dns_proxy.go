package conf

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/cfgcommon"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/dns"
	"github.com/golang/protobuf/proto"
)

type DNSOutboundConfig struct {
	Network cfgcommon.Network  `json:"network"`
	Address *cfgcommon.Address `json:"address"`
	Port    uint16             `json:"port"`
}

func (c *DNSOutboundConfig) Build() (proto.Message, error) {
	config := &dns.Config{
		Server: &net.Endpoint{
			Network: c.Network.Build(),
			Port:    uint32(c.Port),
		},
	}
	if c.Address != nil {
		config.Server.Address = c.Address.Build()
	}
	return config, nil
}
