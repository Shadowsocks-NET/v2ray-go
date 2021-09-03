package transport

import "github.com/Shadowsocks-NET/v2ray-go/v4/common/buf"

// Link is a utility for connecting between an inbound and an outbound proxy handler.
type Link struct {
	Reader buf.Reader
	Writer buf.Writer
}
