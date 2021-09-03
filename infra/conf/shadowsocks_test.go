package conf_test

import (
	"testing"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/protocol"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/serial"
	. "github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/shadowsocks"
)

func TestShadowsocksServerConfigParsing(t *testing.T) {
	creator := func() Buildable {
		return new(ShadowsocksServerConfig)
	}

	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"method": "aes-256-GCM",
				"password": "v2ray-password"
			}`,
			Parser: loadJSON(creator),
			Output: &shadowsocks.ServerConfig{
				User: &protocol.User{
					Account: serial.ToTypedMessage(&shadowsocks.Account{
						CipherType: shadowsocks.CipherType_AES_256_GCM,
						Password:   "v2ray-password",
					}),
				},
				Network: []net.Network{net.Network_TCP},
			},
		},
	})
}
