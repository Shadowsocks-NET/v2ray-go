package internet

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"golang.org/x/sys/unix"
)

const (
	// For incoming connections.
	TCP_FASTOPEN = 23 // nolint: golint,stylecheck
	// For out-going connections.
	TCP_FASTOPEN_CONNECT = 30 // nolint: golint,stylecheck
)

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig, dest net.Destination) error {
	if config.Mark != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, int(config.Mark)); err != nil {
			return newError("failed to set SO_MARK").Base(err)
		}
	}

	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, TCP_FASTOPEN_CONNECT, 1); err != nil {
				return newError("failed to set TCP_FASTOPEN_CONNECT=1").Base(err)
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, TCP_FASTOPEN_CONNECT, 0); err != nil {
				return newError("failed to set TCP_FASTOPEN_CONNECT=0").Base(err)
			}
		}

		if config.TcpKeepAliveInterval != 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, int(config.TcpKeepAliveInterval)); err != nil {
				return newError("failed to set TCP_KEEPINTVL").Base(err)
			}
		}
	}

	if config.Tproxy.IsEnabled() {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_TRANSPARENT, 1); err != nil {
			return newError("failed to set IP_TRANSPARENT").Base(err)
		}
	}

	if config.HasBindInterface() {
		switch network {
		case "tcp4", "tcp6":
			if err := unix.BindToDevice(int(fd), config.BindInterfaceName); err != nil {
				return newError("failed to bind to device ", config.BindInterfaceName).Base(err)
			}

			newError("successfully set SO_BINDTODEVICE to ", dest, " on ifindex ", config.BindInterfaceIndex).AtInfo().WriteToLog()
		}
	}

	return nil
}

func applyInboundSocketOptions(network string, fd uintptr, config *SocketConfig) error {
	if config.Mark != 0 {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, int(config.Mark)); err != nil {
			return newError("failed to set SO_MARK").Base(err)
		}
	}
	if isTCPSocket(network) {
		switch config.Tfo {
		case SocketConfig_Enable:
			if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, TCP_FASTOPEN, 1); err != nil {
				return newError("failed to set TCP_FASTOPEN=1").Base(err)
			}
		case SocketConfig_Disable:
			if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, TCP_FASTOPEN, 0); err != nil {
				return newError("failed to set TCP_FASTOPEN=0").Base(err)
			}
		}

		if config.TcpKeepAliveInterval != 0 {
			if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, int(config.TcpKeepAliveInterval)); err != nil {
				return newError("failed to set TCP_KEEPINTVL", err)
			}
		}
	}

	if config.Tproxy.IsEnabled() {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_TRANSPARENT, 1); err != nil {
			return newError("failed to set IP_TRANSPARENT").Base(err)
		}
	}

	if config.ReceiveOriginalDestAddress && isUDPSocket(network) {
		err1 := unix.SetsockoptInt(int(fd), unix.SOL_IPV6, unix.IPV6_RECVORIGDSTADDR, 1)
		err2 := unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_RECVORIGDSTADDR, 1)
		if err1 != nil && err2 != nil {
			return err1
		}
	}

	return nil
}

func bindAddr(fd uintptr, ip []byte, port uint32) error {
	setReuseAddr(fd)
	setReusePort(fd)

	var sockaddr unix.Sockaddr

	switch len(ip) {
	case net.IPv4len:
		a4 := &unix.SockaddrInet4{
			Port: int(port),
		}
		copy(a4.Addr[:], ip)
		sockaddr = a4
	case net.IPv6len:
		a6 := &unix.SockaddrInet6{
			Port: int(port),
		}
		copy(a6.Addr[:], ip)
		sockaddr = a6
	default:
		return newError("unexpected length of ip")
	}

	return unix.Bind(int(fd), sockaddr)
}

func setReuseAddr(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return newError("failed to set SO_REUSEADDR").Base(err).AtWarning()
	}
	return nil
}

func setReusePort(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return newError("failed to set SO_REUSEPORT").Base(err).AtWarning()
	}
	return nil
}
