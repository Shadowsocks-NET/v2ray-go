package policy_test

import (
	"context"
	"testing"
	"time"

	. "github.com/Shadowsocks-NET/v2ray-go/v4/app/policy"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	"github.com/Shadowsocks-NET/v2ray-go/v4/features/policy"
)

func TestPolicy(t *testing.T) {
	manager, err := New(context.Background(), &Config{
		Level: map[uint32]*Policy{
			0: {
				Timeout: &Policy_Timeout{
					Handshake: &Second{
						Value: 2,
					},
				},
			},
		},
	})
	common.Must(err)

	pDefault := policy.SessionDefault()

	{
		p := manager.ForLevel(0)
		if p.Timeouts.Handshake != 2*time.Second {
			t.Error("expect 2 sec timeout, but got ", p.Timeouts.Handshake)
		}
		if p.Timeouts.UDPIdle != pDefault.Timeouts.UDPIdle {
			t.Error("expect ", pDefault.Timeouts.UDPIdle, " sec timeout, but got ", p.Timeouts.UDPIdle)
		}
	}

	{
		p := manager.ForLevel(1)
		if p.Timeouts.Handshake != pDefault.Timeouts.Handshake {
			t.Error("expect ", pDefault.Timeouts.Handshake, " sec timeout, but got ", p.Timeouts.Handshake)
		}
	}
}
