package internet

import (
	"encoding/binary"
	"unsafe"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"golang.org/x/sys/windows"
)

const (
	TCP_FASTOPEN    = 15 // nolint: golint,stylecheck
	IP_UNICAST_IF   = 31 // nolint: golint,stylecheck
	IPV6_UNICAST_IF = 31 // nolint: golint,stylecheck
)

func setTFO(fd windows.Handle, settings SocketConfig_TCPFastOpenState) error {
	switch settings {
	case SocketConfig_Enable:
		if err := windows.SetsockoptInt(fd, windows.IPPROTO_TCP, TCP_FASTOPEN, 1); err != nil {
			return err
		}
	case SocketConfig_Disable:
		if err := windows.SetsockoptInt(fd, windows.IPPROTO_TCP, TCP_FASTOPEN, 0); err != nil {
			return err
		}
	}
	return nil
}

func applyOutboundSocketOptions(network string, address string, fd uintptr, config *SocketConfig, dest net.Destination) error {
	if isTCPSocket(network) {
		if err := setTFO(windows.Handle(fd), config.Tfo); err != nil {
			return err
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
		if err := setTFO(windows.Handle(fd), config.Tfo); err != nil {
			return err
		}
	}

	if config.BindInterfaceIndex != 0 {
		if err := bindInterface(fd, network, config.BindInterfaceIndex); err != nil {
			return err
		}
	}

	return nil
}

func bindAddr(fd uintptr, ip []byte, port uint32) error {
	return nil
}

func bindInterface(fd uintptr, network string, interfaceIndex uint32) error {
	switch network {
	case "tcp4", "udp4":
		var bytes [4]byte
		binary.BigEndian.PutUint32(bytes[:], interfaceIndex)
		interfaceIndex = *(*uint32)(unsafe.Pointer(&bytes[0]))
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, IP_UNICAST_IF, int(interfaceIndex)); err != nil {
			return err
		}
	case "tcp6", "udp6":
		if err := windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, IPV6_UNICAST_IF, int(interfaceIndex)); err != nil {
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
