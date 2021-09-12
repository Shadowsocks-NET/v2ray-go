package outbound

import (
	"context"

	core "github.com/Shadowsocks-NET/v2ray-go/v4"
	"github.com/Shadowsocks-NET/v2ray-go/v4/app/proxyman"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/mux"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/session"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/outbound"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/policy"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/stats"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/tls"
	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/pipe"
)

func getStatCounter(v *core.Instance, tag string) (stats.Counter, stats.Counter) {
	var uplinkCounter stats.Counter
	var downlinkCounter stats.Counter

	policy := v.GetFeature(policy.ManagerType()).(policy.Manager)
	if len(tag) > 0 && policy.ForSystem().Stats.OutboundUplink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "outbound>>>" + tag + ">>>traffic>>>uplink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			uplinkCounter = c
		}
	}
	if len(tag) > 0 && policy.ForSystem().Stats.OutboundDownlink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "outbound>>>" + tag + ">>>traffic>>>downlink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			downlinkCounter = c
		}
	}

	return uplinkCounter, downlinkCounter
}

// Handler is an implements of outbound.Handler.
type Handler struct {
	tag             string
	senderSettings  *proxyman.SenderConfig
	streamSettings  *internet.MemoryStreamConfig
	proxy           proxy.Outbound
	outboundManager outbound.Manager
	mux             *mux.ClientManager
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter
}

// NewHandler create a new Handler based on the given configuration.
func NewHandler(ctx context.Context, config *core.OutboundHandlerConfig) (outbound.Handler, error) {
	v := core.MustFromContext(ctx)
	uplinkCounter, downlinkCounter := getStatCounter(v, config.Tag)
	h := &Handler{
		tag:             config.Tag,
		outboundManager: v.GetFeature(outbound.ManagerType()).(outbound.Manager),
		uplinkCounter:   uplinkCounter,
		downlinkCounter: downlinkCounter,
	}

	if config.SenderSettings != nil {
		senderSettings, err := config.SenderSettings.GetInstance()
		if err != nil {
			return nil, err
		}
		switch s := senderSettings.(type) {
		case *proxyman.SenderConfig:
			h.senderSettings = s
			mss, err := internet.ToMemoryStreamConfig(s.StreamSettings)
			if err != nil {
				return nil, newError("failed to parse stream settings").Base(err).AtWarning()
			}
			h.streamSettings = mss
		default:
			return nil, newError("settings is not SenderConfig")
		}
	}

	proxyConfig, err := config.ProxySettings.GetInstance()
	if err != nil {
		return nil, err
	}

	rawProxyHandler, err := common.CreateObject(ctx, proxyConfig)
	if err != nil {
		return nil, err
	}

	proxyHandler, ok := rawProxyHandler.(proxy.Outbound)
	if !ok {
		return nil, newError("not an outbound handler")
	}

	if h.senderSettings != nil && h.senderSettings.MultiplexSettings != nil {
		config := h.senderSettings.MultiplexSettings
		if config.Concurrency < 1 || config.Concurrency > 1024 {
			return nil, newError("invalid mux concurrency: ", config.Concurrency).AtWarning()
		}
		h.mux = &mux.ClientManager{
			Enabled: h.senderSettings.MultiplexSettings.Enabled,
			Picker: &mux.IncrementalWorkerPicker{
				Factory: mux.NewDialingWorkerFactory(
					ctx,
					proxyHandler,
					h,
					mux.ClientStrategy{
						MaxConcurrency: config.Concurrency,
						MaxConnection:  128,
					},
				),
			},
		}
	}

	h.proxy = proxyHandler
	return h, nil
}

// Tag implements outbound.Handler.
func (h *Handler) Tag() string {
	return h.tag
}

// Dispatch implements proxy.Outbound.Dispatch.
func (h *Handler) Dispatch(ctx context.Context, link *transport.Link) {
	if h.mux != nil && (h.mux.Enabled || session.MuxPreferedFromContext(ctx)) {
		if err := h.mux.Dispatch(ctx, link); err != nil {
			err := newError("failed to process mux outbound traffic").Base(err)
			session.SubmitOutboundErrorToOriginator(ctx, err)
			err.WriteToLog(session.ExportIDToError(ctx))
			common.Interrupt(link.Writer)
		}
	} else {
		if err := h.proxy.Process(ctx, link, h); err != nil {
			// Ensure outbound ray is properly closed.
			err := newError("failed to process outbound traffic").Base(err)
			session.SubmitOutboundErrorToOriginator(ctx, err)
			err.WriteToLog(session.ExportIDToError(ctx))
			common.Interrupt(link.Writer)
		} else {
			common.Must(common.Close(link.Writer))
		}
		common.Interrupt(link.Reader)
	}
}

// Address implements internet.Dialer.
func (h *Handler) Addresses() (net.Address, net.Address) {
	if h.senderSettings == nil {
		return nil, nil
	}
	return h.senderSettings.Via4.AsAddress(), h.senderSettings.Via6.AsAddress()
}

// Dial implements internet.Dialer.
func (h *Handler) Dial(ctx context.Context, dest net.Destination) (internet.Connection, error) {
	if h.senderSettings != nil {
		if h.senderSettings.ProxySettings.HasTag() {
			if !h.senderSettings.ProxySettings.TransportLayerProxy {
				tag := h.senderSettings.ProxySettings.Tag
				handler := h.outboundManager.GetHandler(tag)
				if handler != nil {
					newError("proxying to ", tag, " for dest ", dest).AtDebug().WriteToLog(session.ExportIDToError(ctx))
					ctx = session.ContextWithOutbound(ctx, &session.Outbound{
						Target: dest,
					})

					opts := pipe.OptionsFromContext(ctx)
					uplinkReader, uplinkWriter := pipe.New(opts...)
					downlinkReader, downlinkWriter := pipe.New(opts...)

					go handler.Dispatch(ctx, &transport.Link{Reader: uplinkReader, Writer: downlinkWriter})
					conn := net.NewConnection(net.ConnectionInputMulti(uplinkWriter), net.ConnectionOutputMulti(downlinkReader))

					if config := tls.ConfigFromStreamSettings(h.streamSettings); config != nil {
						tlsConfig := config.GetTLSConfig(tls.WithDestination(dest))
						conn = tls.Client(conn, tlsConfig)
					}

					return h.getStatCouterConnection(conn), nil
				}

				newError("failed to get outbound handler with tag: ", tag).AtWarning().WriteToLog(session.ExportIDToError(ctx))
			} else {
				tag := h.senderSettings.ProxySettings.Tag
				newError("transport layer proxying to ", tag, " for dest ", dest).AtDebug().WriteToLog(session.ExportIDToError(ctx))
				ctx = session.SetTransportLayerProxyTagToContext(ctx, tag)
			}
		}

		if h.senderSettings.IsAdvanced() {
			outbound := session.OutboundFromContext(ctx)
			if outbound == nil {
				outbound = new(session.Outbound)
				ctx = session.ContextWithOutbound(ctx, outbound)
			}
			outbound.Bind4 = h.senderSettings.Via4.AsAddress()
			outbound.Bind6 = h.senderSettings.Via6.AsAddress()
			outbound.DomainStrategy = int32(h.senderSettings.DomainStrategy)
			outbound.FallbackDelayMs = h.senderSettings.FallbackDelayMs
		}
	}

	conn, err := internet.Dial(ctx, dest, h.streamSettings)
	return h.getStatCouterConnection(conn), err
}

func (h *Handler) getStatCouterConnection(conn internet.Connection) internet.Connection {
	if h.uplinkCounter != nil || h.downlinkCounter != nil {
		return &internet.StatCouterConnection{
			Connection:   conn,
			ReadCounter:  h.downlinkCounter,
			WriteCounter: h.uplinkCounter,
		}
	}
	return conn
}

// GetOutbound implements proxy.GetOutbound.
func (h *Handler) GetOutbound() proxy.Outbound {
	return h.proxy
}

// Start implements common.Runnable.
func (h *Handler) Start() error {
	return nil
}

// Close implements common.Closable.
func (h *Handler) Close() error {
	common.Close(h.mux)
	return nil
}
