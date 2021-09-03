package udp

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/buf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
)

// Packet is a UDP packet together with its source and destination address.
type Packet struct {
	Payload *buf.Buffer
	Source  net.Destination
	Target  net.Destination
}
