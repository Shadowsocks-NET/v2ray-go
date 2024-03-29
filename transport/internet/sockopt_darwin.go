package internet

import (
	"golang.org/x/sys/unix"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
)

const (
	// TCP_FASTOPEN is the socket option on darwin for TCP fast open.
	TCP_FASTOPEN = 0x105 // nolint: golint,stylecheck
	// TCP_FASTOPEN_SERVER is the value to enable TCP fast open on darwin for server connections.
	TCP_FASTOPEN_SERVER = 0x01 // nolint: golint,stylecheck
	// TCP_FASTOPEN_CLIENT is the value to enable TCP fast open on darwin for client connections.
	TCP_FASTOPEN_CLIENT = 0x02 // nolint: golint,stylecheck
)

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig, dest net.Destination) error {
	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, TCP_FASTOPEN, TCP_FASTOPEN_CLIENT); err != nil {
				return err
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, TCP_FASTOPEN, 0); err != nil {
				return err
			}
		}
	}

	if config.BindInterfaceIndex != 0 {
		if err := bindInterface(fd, network, config.BindInterfaceIndex); err != nil {
			return err
		}
	}

	return nil
}

func applyInboundSocketOptions(network string, fd uintptr, config *SocketConfig) error {
	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, TCP_FASTOPEN, TCP_FASTOPEN_SERVER); err != nil {
				return err
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, TCP_FASTOPEN, 0); err != nil {
				return err
			}
		}
	}

	if config.BindInterfaceIndex != 0 {
		if err := bindInterface(fd, network, config.BindInterfaceIndex); err != nil {
			return err
		}
	}

	return nil
}

func bindAddr(fd uintptr, address []byte, port uint32) error {
	return nil
}

func bindInterface(fd uintptr, network string, interfaceIndex uint32) error {
	switch network {
	case "tcp4", "udp4":
		if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, int(interfaceIndex)); err != nil {
			return err
		}
	case "tcp6", "udp6":
		if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, int(interfaceIndex)); err != nil {
			return err
		}
	}

	return nil
}

func setReuseAddr(fd uintptr) error {
	return nil
}

func setReusePort(fd uintptr) error {
	return nil
}
