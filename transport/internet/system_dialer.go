package internet

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/net"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/session"
)

var effectiveSystemDialer SystemDialer = &DefaultSystemDialer{}

type SystemDialer interface {
	Dial(ctx context.Context, source net.Address, destination net.Destination, sockopt *SocketConfig) (net.Conn, error)
	DialIPs(ctx context.Context, src4 net.Address, dests4 []net.Destination, src6 net.Address, dests6 []net.Destination, sockopt *SocketConfig) (net.Conn, error)
	GetFallbackDelay() time.Duration
	SetFallbackDelay(time.Duration)
}

type DefaultSystemDialer struct {
	controllers []controller

	// FallbackDelay specifies the length of time to wait before
	// spawning a RFC 6555 Fast Fallback connection. That is, this
	// is the amount of time to wait for IPv6 to succeed before
	// assuming that IPv6 is misconfigured and falling back to
	// IPv4.
	//
	// If zero, a default delay of 300ms is used.
	// A negative value disables Fast Fallback support.
	FallbackDelay time.Duration
}

func resolveSrcAddr(network net.Network, src net.Address) net.Addr {
	if src == nil || src == net.AnyIP {
		return nil
	}

	if network == net.Network_TCP {
		return &net.TCPAddr{
			IP:   src.IP(),
			Port: 0,
		}
	}

	return &net.UDPAddr{
		IP:   src.IP(),
		Port: 0,
	}
}

func (d *DefaultSystemDialer) GetFallbackDelay() time.Duration {
	return d.FallbackDelay
}

func (d *DefaultSystemDialer) SetFallbackDelay(t time.Duration) {
	d.FallbackDelay = t
}

func (d *DefaultSystemDialer) Dial(ctx context.Context, src net.Address, dest net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	if dest.Network == net.Network_UDP && !sockopt.HasBindAddr() {
		srcAddr := resolveSrcAddr(net.Network_UDP, src)
		var destAddr *net.UDPAddr
		addressFamily := dest.Address.Family()
		var bindInterfaceIp4, bindInterfaceIp6 []byte

		if sockopt.HasBindInterface() {
			bindInterfaceIp4 = sockopt.BindInterfaceIp4
			bindInterfaceIp6 = sockopt.BindInterfaceIp6
		} else {
			bindInterfaceIp4 = make([]byte, 4)
			bindInterfaceIp6 = make([]byte, 16)
		}

		switch addressFamily {
		case net.AddressFamilyDomain:
			if srcAddr == nil {
				srcAddr = &net.UDPAddr{
					IP:   bindInterfaceIp4,
					Port: 0,
				}
			}
			var err error
			destAddr, err = net.ResolveUDPAddr("udp", dest.NetAddr())
			if err != nil {
				return nil, err
			}
			if destAddr.IP.To4() == nil {
				addressFamily = net.AddressFamilyIPv6
			} else {
				addressFamily = net.AddressFamilyIPv4
			}

		case net.AddressFamilyIPv4:
			if srcAddr == nil {
				srcAddr = &net.UDPAddr{
					IP:   bindInterfaceIp4,
					Port: 0,
				}
			}
			destAddr = &net.UDPAddr{
				IP:   dest.Address.IP(),
				Port: int(dest.Port),
			}

		case net.AddressFamilyIPv6:
			if srcAddr == nil {
				srcAddr = &net.UDPAddr{
					IP:   bindInterfaceIp6,
					Port: 0,
				}
			}
			destAddr = &net.UDPAddr{
				IP:   dest.Address.IP(),
				Port: int(dest.Port),
			}
		}

		packetConn, err := ListenSystemPacket(ctx, srcAddr, sockopt)
		if err != nil {
			return nil, err
		}

		return newUdpConnWrapper(packetConn.(*net.UDPConn), src, destAddr, addressFamily, sockopt)
	}

	dialer := &net.Dialer{
		Timeout:   time.Second * 16,
		LocalAddr: resolveSrcAddr(dest.Network, src),
	}

	if sockopt != nil || len(d.controllers) > 0 {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if sockopt != nil {
					if err := applyOutboundSocketOptions(network, address, fd, sockopt, dest); err != nil {
						newError("failed to apply socket options").Base(err).WriteToLog(session.ExportIDToError(ctx))
					}
					if dest.Network == net.Network_UDP && sockopt.HasBindAddr() {
						if err := bindAddr(fd, sockopt.BindAddress, sockopt.BindPort); err != nil {
							newError("failed to bind source address to ", sockopt.BindAddress).Base(err).WriteToLog(session.ExportIDToError(ctx))
						}
					}
				}

				for _, ctl := range d.controllers {
					if err := ctl(network, address, fd); err != nil {
						newError("failed to apply external controller").Base(err).WriteToLog(session.ExportIDToError(ctx))
					}
				}
			})
		}
	}

	return dialer.DialContext(ctx, dest.Network.SystemString(), dest.NetAddr())
}

func (d *DefaultSystemDialer) DialIPs(ctx context.Context, src4 net.Address, dests4 []net.Destination, src6 net.Address, dests6 []net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	if ctx == nil {
		panic("nil context")
	}

	dial4 := len(dests4) > 0
	dial6 := len(dests6) > 0

	if dial4 && dial6 {
		if d.FallbackDelay >= 0 {
			return d.DialParallel(ctx, src6, dests6, src4, dests4, sockopt)
		} else if src4 == nil && src6 == nil { // no custom local address. prefer v6.
			dests := append(dests6, dests4...)
			return d.DialSerial(ctx, nil, dests, sockopt)
		} else { // has custom local address. dial v6 only.
			return d.DialSerial(ctx, src6, dests6, sockopt)
		}
	} else if dial4 {
		return d.DialSerial(ctx, src4, dests4, sockopt)
	} else if dial6 {
		return d.DialSerial(ctx, src6, dests6, sockopt)
	} else {
		return nil, newError("failed to dial: missing address")
	}
}

// DialParallel races two copies of dialSerial, giving the first a
// head start. It returns the first established connection and
// closes the others. Otherwise it returns an error from the first
// primary address.
func (d *DefaultSystemDialer) DialParallel(ctx context.Context, srcp net.Address, primaries []net.Destination, srcf net.Address, fallbacks []net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	if len(fallbacks) == 0 {
		return d.DialSerial(ctx, srcp, primaries, sockopt)
	}

	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		primary bool
		done    bool
	}
	results := make(chan dialResult) // unbuffered

	startRacer := func(ctx context.Context, primary bool) {
		las := srcp
		ras := primaries
		if !primary {
			las = srcf
			ras = fallbacks
		}
		c, err := d.DialSerial(ctx, las, ras, sockopt)
		select {
		case results <- dialResult{Conn: c, error: err, primary: primary, done: true}:
		case <-returned:
			if c != nil {
				c.Close()
			}
		}
	}

	var primary, fallback dialResult

	// Start the main racer.
	primaryCtx, primaryCancel := context.WithCancel(ctx)
	defer primaryCancel()
	go startRacer(primaryCtx, true)

	// Start the timer for the fallback racer.
	fallbackTimer := time.NewTimer(d.fallbackDelay())
	defer fallbackTimer.Stop()

	for {
		select {
		case <-fallbackTimer.C:
			fallbackCtx, fallbackCancel := context.WithCancel(ctx)
			defer fallbackCancel()
			go startRacer(fallbackCtx, false)

		case res := <-results:
			if res.error == nil {
				return res.Conn, nil
			}
			if res.primary {
				primary = res
			} else {
				fallback = res
			}
			if primary.done && fallback.done {
				return nil, primary.error
			}
			if res.primary && fallbackTimer.Stop() {
				// If we were able to stop the timer, that means it
				// was running (hadn't yet started the fallback), but
				// we just got an error on the primary path, so start
				// the fallback immediately (in 0 nanoseconds).
				fallbackTimer.Reset(0)
			}
		}
	}
}

// DialSerial connects to a list of addresses in sequence, returning
// either the first successful connection, or the first error.
func (d *DefaultSystemDialer) DialSerial(ctx context.Context, src net.Address, dests []net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	var firstErr error // The error from the first address is most relevant.

	for i, dest := range dests {
		select {
		case <-ctx.Done():
			return nil, newError("failed to dial ", dest).Base(ctx.Err())
		default:
		}

		dialCtx := ctx
		if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
			partialDeadline, err := partialDeadline(time.Now(), deadline, len(dests)-i)
			if err != nil {
				// Ran out of time.
				if firstErr == nil {
					firstErr = newError("failed to dial ", dest).Base(err)
				}
				break
			}
			if partialDeadline.Before(deadline) {
				var cancel context.CancelFunc
				dialCtx, cancel = context.WithDeadline(ctx, partialDeadline)
				defer cancel()
			}
		}

		c, err := d.Dial(dialCtx, src, dest, sockopt)
		if err == nil {
			return c, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}

	if firstErr == nil {
		firstErr = newError("failed to dial: missing address")
	}
	return nil, firstErr
}

// partialDeadline returns the deadline to use for a single address,
// when multiple addresses are pending.
func partialDeadline(now, deadline time.Time, addrsRemaining int) (time.Time, error) {
	if deadline.IsZero() {
		return deadline, nil
	}
	timeRemaining := deadline.Sub(now)
	if timeRemaining <= 0 {
		return time.Time{}, os.ErrDeadlineExceeded
	}
	// Tentatively allocate equal time to each remaining address.
	timeout := timeRemaining / time.Duration(addrsRemaining)
	// If the time per address is too short, steal from the end of the list.
	const saneMinimum = 2 * time.Second
	if timeout < saneMinimum {
		if timeRemaining < saneMinimum {
			timeout = timeRemaining
		} else {
			timeout = saneMinimum
		}
	}
	return now.Add(timeout), nil
}

// fallbackDelay gets the fallback delay in effect.
func (d *DefaultSystemDialer) fallbackDelay() time.Duration {
	if d.FallbackDelay > 0 {
		return d.FallbackDelay
	} else {
		return 300 * time.Millisecond
	}
}

type udpConnWrapper struct {
	conn *net.UDPConn
	oob  []byte
	da   *net.UDPAddr
}

func (c *udpConnWrapper) Close() error {
	return c.conn.Close()
}

func (c *udpConnWrapper) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *udpConnWrapper) RemoteAddr() net.Addr {
	return c.da
}

func (c *udpConnWrapper) Write(p []byte) (int, error) {
	n, _, err := c.conn.WriteMsgUDP(p, c.oob, c.da)
	return n, err
}

func (c *udpConnWrapper) Read(p []byte) (int, error) {
	return c.conn.Read(p)
}

func (c *udpConnWrapper) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *udpConnWrapper) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *udpConnWrapper) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type SystemDialerAdapter interface {
	Dial(network string, address string) (net.Conn, error)
}

type SimpleSystemDialer struct {
	adapter SystemDialerAdapter
}

func WithAdapter(dialer SystemDialerAdapter) SystemDialer {
	return &SimpleSystemDialer{
		adapter: dialer,
	}
}

func (v *SimpleSystemDialer) GetFallbackDelay() time.Duration {
	return 300 * time.Millisecond
}

func (v *SimpleSystemDialer) SetFallbackDelay(t time.Duration) {
}

func (v *SimpleSystemDialer) Dial(ctx context.Context, src net.Address, dest net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	return v.adapter.Dial(dest.Network.SystemString(), dest.NetAddr())
}

func (v *SimpleSystemDialer) DialIPs(ctx context.Context, src4 net.Address, dests4 []net.Destination, src6 net.Address, dests6 []net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	if len(dests6) > 0 {
		dest := dests6[0]
		return v.adapter.Dial(dest.Network.SystemString(), dest.NetAddr())
	} else if len(dests4) > 0 {
		dest := dests4[0]
		return v.adapter.Dial(dest.Network.SystemString(), dest.NetAddr())
	} else {
		return nil, newError("empty destination")
	}
}

// UseAlternativeSystemDialer replaces the current system dialer with a given one.
// Caller must ensure there is no race condition.
//
// v2ray:api:stable
func UseAlternativeSystemDialer(dialer SystemDialer) {
	if dialer == nil {
		dialer = &DefaultSystemDialer{}
	}
	effectiveSystemDialer = dialer
}

// RegisterDialerController adds a controller to the effective system dialer.
// The controller can be used to operate on file descriptors before they are put into use.
// It only works when effective dialer is the default dialer.
//
// v2ray:api:beta
func RegisterDialerController(ctl func(network, address string, fd uintptr) error) error {
	if ctl == nil {
		return newError("nil listener controller")
	}

	dialer, ok := effectiveSystemDialer.(*DefaultSystemDialer)
	if !ok {
		return newError("RegisterListenerController not supported in custom dialer")
	}

	dialer.controllers = append(dialer.controllers, ctl)
	return nil
}
