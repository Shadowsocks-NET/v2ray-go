package conf_test

import (
	"testing"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/serial"
	. "github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/blackhole"
)

func TestHTTPResponseJSON(t *testing.T) {
	creator := func() Buildable {
		return new(BlackholeConfig)
	}

	runMultiTestCase(t, []TestCase{
		{
			Input: `{
				"response": {
					"type": "http"
				}
			}`,
			Parser: loadJSON(creator),
			Output: &blackhole.Config{
				Response: serial.ToTypedMessage(&blackhole.HTTPResponse{}),
			},
		},
		{
			Input:  `{}`,
			Parser: loadJSON(creator),
			Output: &blackhole.Config{},
		},
	})
}
