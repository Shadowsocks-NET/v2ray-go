package transport

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
)

// Apply applies this Config.
func (c *Config) Apply() error {
	if c == nil {
		return nil
	}
	return internet.ApplyGlobalTransportSettings(c.TransportSettings)
}
