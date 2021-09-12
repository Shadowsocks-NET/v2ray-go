//go:build !linux
// +build !linux

package internet

import "github.com/Shadowsocks-NET/v2ray-go/v4/common/net"

func newUdpConnWrapper(conn *net.UDPConn, src net.Address, destAddr *net.UDPAddr, addressFamily net.AddressFamily, sockopt *SocketConfig) (*udpConnWrapper, error) {
	return &udpConnWrapper{
		conn: conn,
		da:   destAddr,
	}, nil
}
