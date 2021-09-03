package dns

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features"
)

type FakeDNSEngine interface {
	features.Feature
	GetFakeIPForDomain(domain string) []net.Address
	GetDomainFromFakeDNS(ip net.Address) string
}

type FakeDNSEngineRev0 interface {
	FakeDNSEngine
	IsIPInIPPool(ip net.Address) bool
	GetFakeIPForDomain3(domain string, IPv4, IPv6 bool) []net.Address
}
