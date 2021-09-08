//go:build !confonly
// +build !confonly

package freedom

//go:generate go run github.com/Shadowsocks-NET/v2ray-go/v4/common/errors/errorgen

import (
	"context"

	core "github.com/Shadowsocks-NET/v2ray-go/v4"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/buf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/retry"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/session"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/signal"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/task"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/policy"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		h := new(Handler)
		if err := core.RequireFeatures(ctx, func(pm policy.Manager) error {
			return h.Init(config.(*Config), pm)
		}); err != nil {
			return nil, err
		}
		return h, nil
	}))
}

// Handler handles Freedom connections.
type Handler struct {
	policyManager policy.Manager
	config        *Config
}

// Init initializes the Handler with necessary parameters.
func (h *Handler) Init(config *Config, pm policy.Manager) error {
	h.config = config
	h.policyManager = pm

	return nil
}

func isValidAddress(addr *net.IPOrDomain) bool {
	if addr == nil {
		return false
	}

	a := addr.AsAddress()
	return a != net.AnyIP
}

// Process implements proxy.Outbound.
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified.")
	}
	destination := outbound.Target
	if h.config.DestinationOverride != nil {
		server := h.config.DestinationOverride.Server
		if isValidAddress(server.Address) {
			destination.Address = server.Address.AsAddress()
		}
		if server.Port != 0 {
			destination.Port = net.Port(server.Port)
		}
	}
	newError("opening connection to ", destination).WriteToLog(session.ExportIDToError(ctx))

	input := link.Reader
	output := link.Writer

	var conn internet.Connection
	err := retry.ExponentialBackoff(5, 100).On(func() error {
		rawConn, err := dialer.Dial(ctx, destination)
		if err != nil {
			return err
		}
		conn = rawConn
		return nil
	})
	if err != nil {
		return newError("failed to open connection to ", destination).Base(err)
	}
	defer conn.Close()

	plcy := h.policyManager.ForLevel(h.config.UserLevel)
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, plcy.Timeouts.ConnectionIdle)

	requestDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.DownlinkOnly)

		var writer buf.Writer
		if destination.Network == net.Network_TCP {
			writer = buf.NewWriter(conn)
		} else {
			writer = &buf.SequentialWriter{Writer: conn}
		}

		if err := buf.Copy(input, writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to process request").Base(err)
		}

		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.UplinkOnly)

		var reader buf.Reader
		if destination.Network == net.Network_TCP {
			reader = buf.NewReader(conn)
		} else {
			reader = buf.NewPacketReader(conn)
		}
		if err := buf.Copy(reader, output, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to process response").Base(err)
		}

		return nil
	}

	if err := task.Run(ctx, requestDone, task.OnSuccess(responseDone, task.Close(output))); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}
