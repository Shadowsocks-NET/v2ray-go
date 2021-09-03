package tagged

import (
	"context"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
)

type DialFunc func(ctx context.Context, dest net.Destination, tag string) (net.Conn, error)

var Dialer DialFunc
