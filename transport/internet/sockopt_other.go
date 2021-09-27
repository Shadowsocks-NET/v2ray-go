//go:build js || dragonfly || netbsd || openbsd || solaris
// +build js dragonfly netbsd openbsd solaris

package internet

import "github.com/Shadowsocks-NET/v2ray-go/v4/common/net"

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig, dest net.Destination) error {
	return nil
}

func applyInboundSocketOptions(network string, fd uintptr, config *SocketConfig) error {
	return nil
}

func bindAddr(fd uintptr, ip []byte, port uint32) error {
	return nil
}

func setReuseAddr(fd uintptr) error {
	return nil
}

func setReusePort(fd uintptr) error {
	return nil
}
