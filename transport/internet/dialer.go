package internet

import (
	"context"
	"time"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/session"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/dns"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/tagged"
)

// Dialer is the interface for dialing outbound connections.
type Dialer interface {
	// Dial dials a system connection to the given destination.
	Dial(ctx context.Context, destination net.Destination) (Connection, error)

	// Addresses returns the local IPv4 and IPv6 addresses used by this Dialer. Maybe nil if not known.
	Addresses() (addr4, addr6 net.Address)
}

// dialFunc is an interface to dial network connection to a specific destination.
type dialFunc func(ctx context.Context, dest net.Destination, streamSettings *MemoryStreamConfig) (Connection, error)

var (
	transportDialerCache = make(map[string]dialFunc)
	DialerDNSClient      dns.Client //nolint: golint,stylecheck
)

// RegisterTransportDialer registers a Dialer with given name.
func RegisterTransportDialer(protocol string, dialer dialFunc) error {
	if _, found := transportDialerCache[protocol]; found {
		return newError(protocol, " dialer already registered").AtError()
	}
	transportDialerCache[protocol] = dialer
	return nil
}

// Dial dials a internet connection towards the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *MemoryStreamConfig) (Connection, error) {
	if dest.Network == net.Network_TCP {
		if streamSettings == nil {
			s, err := ToMemoryStreamConfig(nil)
			if err != nil {
				return nil, newError("failed to create default stream settings").Base(err)
			}
			streamSettings = s
		}

		protocol := streamSettings.ProtocolName
		dialer := transportDialerCache[protocol]
		if dialer == nil {
			return nil, newError(protocol, " dialer not registered").AtError()
		}
		return dialer(ctx, dest, streamSettings)
	}

	if dest.Network == net.Network_UDP {
		udpDialer := transportDialerCache["udp"]
		if udpDialer == nil {
			return nil, newError("UDP dialer not registered").AtError()
		}
		return udpDialer(ctx, dest, streamSettings)
	}

	return nil, newError("unknown network ", dest.Network)
}

// DialSystem calls system dialer to create a network connection.
func DialSystem(ctx context.Context, dest net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	var la4, la6 net.Address
	var domainStrategy int32
	var fallbackDelay time.Duration

	if outbound := session.OutboundFromContext(ctx); outbound != nil {
		la4 = outbound.Bind4
		la6 = outbound.Bind6
		domainStrategy = outbound.DomainStrategy
		fallbackDelay = time.Duration(outbound.FallbackDelayMs) * time.Millisecond
	}

	if transportLayerOutgoingTag := session.GetTransportLayerProxyTagFromContext(ctx); transportLayerOutgoingTag != "" {
		return DialTaggedOutbound(ctx, dest, transportLayerOutgoingTag)
	}

	effectiveSystemDialer.SetFallbackDelay(fallbackDelay)
	newError("Set fallback delay to ", fallbackDelay).AtDebug().WriteToLog()

	// If the outbound is bound to an interface, we have to make sure the destination address
	// is resolved to an IP. The AsIs domain strategy is overridden by BindInterfaceIndex.
	if domainStrategy == 0 && la4 == nil && la6 == nil && !sockopt.HasBindInterface() {
		return effectiveSystemDialer.Dial(ctx, nil, dest, sockopt)
	}

	ips4, ips6 := resolveIP(ctx, domainStrategy, dest.Address)
	var dests4, dests6 []net.Destination

	for _, ip4 := range ips4 {
		dests4 = append(dests4, net.Destination{
			Address: net.IPAddress(ip4),
			Port:    dest.Port,
			Network: dest.Network,
		})
	}

	for _, ip6 := range ips6 {
		dests6 = append(dests6, net.Destination{
			Address: net.IPAddress(ip6),
			Port:    dest.Port,
			Network: dest.Network,
		})
	}

	return effectiveSystemDialer.DialIPs(ctx, la4, dests4, la6, dests6, sockopt)
}

func DialTaggedOutbound(ctx context.Context, dest net.Destination, tag string) (net.Conn, error) {
	if tagged.Dialer == nil {
		return nil, newError("tagged dial not enabled")
	}
	return tagged.Dialer(ctx, dest, tag)
}

func resolveIP(ctx context.Context, domainStrategy int32, address net.Address) (ips4, ips6 []net.IP) {
	newError("resolveIP processing ", address).AtDebug().WriteToLog()
	if DialerDNSClient == nil {
		newError("DNS client is nil").AtError().WriteToLog()
		return
	}

	if address.Family().IsIP() {
		newError(address, " is IP").AtDebug().WriteToLog()
		ip := address.IP()
		if ip.To4() == nil {
			ips6 = append(ips6, ip)
		} else {
			ips4 = append(ips4, ip)
		}
		return
	}

	domain := address.Domain()

	if c, ok := DialerDNSClient.(dns.ClientWithIPOption); ok {
		c.SetFakeDNSOption(false) // Skip FakeDNS
	} else {
		newError("DNS client doesn't implement ClientWithIPOption")
	}

	var err error
	switch domainStrategy {
	case 0, 1:
		var ips []net.IP
		ips, err = DialerDNSClient.LookupIP(domain)
		for _, ip := range ips {
			if ip.To4() == nil {
				ips6 = append(ips6, ip)
			} else {
				ips4 = append(ips4, ip)
			}
		}
	case 2:
		ips4, err = DialerDNSClient.(dns.IPv4Lookup).LookupIPv4(domain)
	case 3:
		ips6, err = DialerDNSClient.(dns.IPv6Lookup).LookupIPv6(domain)
	}

	if err != nil {
		newError("failed to get IP address for domain ", domain).Base(err).WriteToLog(session.ExportIDToError(ctx))
	}

	return //nolint: nakedret
}
