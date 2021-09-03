//go:build !linux && !freebsd && !confonly
// +build !linux,!freebsd,!confonly

package tcp

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
)

func GetOriginalDestination(conn internet.Connection) (net.Destination, error) {
	return net.Destination{}, nil
}
