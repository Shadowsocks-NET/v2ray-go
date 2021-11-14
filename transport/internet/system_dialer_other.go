//go:build !linux
// +build !linux

package internet

import "github.com/Shadowsocks-NET/v2ray-go/v4/common/net"

func newUDPConnWrapper(conn *net.UDPConn, destAddr *net.UDPAddr, addressFamily net.AddressFamily, sockopt *SocketConfig) (*udpConnWrapper, error) {
	return &udpConnWrapper{
		UDPConn: conn,
		da:      destAddr,
	}, nil
}

func (sockopt *SocketConfig) getBindInterfaceIP46() (bindInterfaceIP4, bindInterfaceIP6 []byte) {
	bindInterfaceIP4 = make([]byte, 4)
	bindInterfaceIP6 = make([]byte, 16)
	return
}
