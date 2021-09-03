//go:build !confonly
// +build !confonly

package tcp

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
)

const protocolName = "tcp"

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
