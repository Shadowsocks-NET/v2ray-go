//go:build !confonly
// +build !confonly

package outbound

//go:generate go run github.com/Shadowsocks-NET/v2ray-go/v4/common/errors/errorgen

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"hash/crc64"
	"time"

	core "github.com/Shadowsocks-NET/v2ray-go/v4"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/buf"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/platform"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/protocol"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/retry"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/session"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/signal"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/task"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/policy"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/encoding"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
)

// Handler is an outbound connection handler for VMess protocol.
type Handler struct {
	server        *protocol.ServerSpec
	policyManager policy.Manager
}

// New creates a new VMess outbound handler.
func New(ctx context.Context, config *Config) (*Handler, error) {
	server, err := protocol.NewServerSpecFromPB(config.Server)
	if err != nil {
		return nil, newError("failed to parse server spec").Base(err)
	}

	v := core.MustFromContext(ctx)
	handler := &Handler{
		server:        server,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
	}

	return handler, nil
}

// Process implements proxy.Outbound.Process().
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	var conn internet.Connection

	err := retry.ExponentialBackoff(5, 200).On(func() error {
		rawConn, err := dialer.Dial(ctx, h.server.Destination())
		if err != nil {
			return err
		}
		conn = rawConn

		return nil
	})
	if err != nil {
		return newError("failed to find an available destination").Base(err).AtWarning()
	}
	defer conn.Close()

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified").AtError()
	}

	target := outbound.Target
	newError("tunneling request to ", target, " via ", h.server.Destination()).WriteToLog(session.ExportIDToError(ctx))

	command := protocol.RequestCommandTCP
	if target.Network == net.Network_UDP {
		command = protocol.RequestCommandUDP
	}
	if target.Address.Family().IsDomain() && target.Address.Domain() == "v1.mux.cool" {
		command = protocol.RequestCommandMux
	}

	user := h.server.PickUser()
	request := &protocol.RequestHeader{
		Version: encoding.Version,
		User:    user,
		Command: command,
		Address: target.Address,
		Port:    target.Port,
		Option:  protocol.RequestOptionChunkStream,
	}

	account := request.User.Account.(*vmess.MemoryAccount)
	request.Security = account.Security

	if request.Security == protocol.SecurityType_AES128_GCM || request.Security == protocol.SecurityType_NONE || request.Security == protocol.SecurityType_CHACHA20_POLY1305 {
		request.Option.Set(protocol.RequestOptionChunkMasking)
	}

	if shouldEnablePadding(request.Security) && request.Option.Has(protocol.RequestOptionChunkMasking) {
		request.Option.Set(protocol.RequestOptionGlobalPadding)
	}

	if request.Security == protocol.SecurityType_ZERO {
		request.Security = protocol.SecurityType_NONE
		request.Option.Clear(protocol.RequestOptionChunkStream)
		request.Option.Clear(protocol.RequestOptionChunkMasking)
	}

	if account.AuthenticatedLengthExperiment {
		request.Option.Set(protocol.RequestOptionAuthenticatedLength)
	}

	input := link.Reader
	output := link.Writer

	hashkdf := hmac.New(sha256.New, []byte("VMessBF"))
	hashkdf.Write(account.ID.Bytes())

	behaviorSeed := crc64.Checksum(hashkdf.Sum(nil), crc64.MakeTable(crc64.ISO))

	session := encoding.NewClientSession(ctx, protocol.DefaultIDHash, int64(behaviorSeed))
	sessionPolicy := h.policyManager.ForLevel(request.User.Level)

	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)

	requestDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)

		writer := buf.NewBufferedWriter(buf.NewWriter(conn))
		if err := session.EncodeRequestHeader(request, writer); err != nil {
			return newError("failed to encode request").Base(err).AtWarning()
		}

		bodyWriter := session.EncodeRequestBody(request, writer)
		if err := buf.CopyOnceTimeout(input, bodyWriter, time.Millisecond*100); err != nil && err != buf.ErrNotTimeoutReader && err != buf.ErrReadTimeout {
			return newError("failed to write first payload").Base(err)
		}

		if err := writer.SetBuffered(false); err != nil {
			return err
		}

		if err := buf.Copy(input, bodyWriter, buf.UpdateActivity(timer)); err != nil {
			return err
		}

		if request.Option.Has(protocol.RequestOptionChunkStream) && !account.NoTerminationSignal {
			if err := bodyWriter.WriteMultiBuffer(buf.MultiBuffer{}); err != nil {
				return err
			}
		}

		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

		reader := &buf.BufferedReader{Reader: buf.NewReader(conn)}
		header, err := session.DecodeResponseHeader(reader)
		if err != nil {
			return newError("failed to read header").Base(err)
		}
		h.handleCommand(h.server.Destination(), header.Command)

		bodyReader := session.DecodeResponseBody(request, reader)

		return buf.Copy(bodyReader, output, buf.UpdateActivity(timer))
	}

	responseDonePost := task.OnSuccess(responseDone, task.Close(output))
	if err := task.Run(ctx, requestDone, responseDonePost); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

var enablePadding = false

func shouldEnablePadding(s protocol.SecurityType) bool {
	return enablePadding || s == protocol.SecurityType_AES128_GCM || s == protocol.SecurityType_CHACHA20_POLY1305 || s == protocol.SecurityType_AUTO
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))

	const defaultFlagValue = "NOT_DEFINED_AT_ALL"

	paddingValue := platform.NewEnvFlag("v2ray.vmess.padding").GetValue(func() string { return defaultFlagValue })
	if paddingValue != defaultFlagValue {
		enablePadding = true
	}
}
