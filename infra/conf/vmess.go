package conf

import (
	"encoding/json"
	"strings"

	"github.com/Shadowsocks-NET/v2ray-go/v4/common/protocol"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/serial"
	"github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/cfgcommon"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/inbound"
	"github.com/Shadowsocks-NET/v2ray-go/v4/proxy/vmess/outbound"
	"github.com/golang/protobuf/proto"
)

type VMessAccount struct {
	ID          string `json:"id"`
	Security    string `json:"security"`
	Experiments string `json:"experiments"`
}

// Build implements Buildable
func (a *VMessAccount) Build() *vmess.Account {
	var st protocol.SecurityType
	switch strings.ToLower(a.Security) {
	case "aes-128-gcm":
		st = protocol.SecurityType_AES128_GCM
	case "chacha20-poly1305":
		st = protocol.SecurityType_CHACHA20_POLY1305
	case "auto":
		st = protocol.SecurityType_AUTO
	case "none":
		st = protocol.SecurityType_NONE
	case "zero":
		st = protocol.SecurityType_ZERO
	default:
		st = protocol.SecurityType_AUTO
	}
	return &vmess.Account{
		Id: a.ID,
		SecuritySettings: &protocol.SecurityConfig{
			Type: st,
		},
		TestsEnabled: a.Experiments,
	}
}

type VMessDetourConfig struct {
	ToTag string `json:"to"`
}

// Build implements Buildable
func (c *VMessDetourConfig) Build() *inbound.DetourConfig {
	return &inbound.DetourConfig{
		To: c.ToTag,
	}
}

type FeaturesConfig struct {
	Detour *VMessDetourConfig `json:"detour"`
}

type VMessDefaultConfig struct {
	Level byte `json:"level"`
}

// Build implements Buildable
func (c *VMessDefaultConfig) Build() *inbound.DefaultConfig {
	config := new(inbound.DefaultConfig)
	config.Level = uint32(c.Level)
	return config
}

type VMessInboundConfig struct {
	Users        []json.RawMessage   `json:"clients"`
	Features     *FeaturesConfig     `json:"features"`
	Defaults     *VMessDefaultConfig `json:"default"`
	DetourConfig *VMessDetourConfig  `json:"detour"`
	SecureOnly   bool                `json:"disableInsecureEncryption"`
}

// Build implements Buildable
func (c *VMessInboundConfig) Build() (proto.Message, error) {
	config := &inbound.Config{
		SecureEncryptionOnly: c.SecureOnly,
	}

	if c.Defaults != nil {
		config.Default = c.Defaults.Build()
	}

	if c.DetourConfig != nil {
		config.Detour = c.DetourConfig.Build()
	} else if c.Features != nil && c.Features.Detour != nil {
		config.Detour = c.Features.Detour.Build()
	}

	config.User = make([]*protocol.User, len(c.Users))
	for idx, rawData := range c.Users {
		user := new(protocol.User)
		if err := json.Unmarshal(rawData, user); err != nil {
			return nil, newError("invalid VMess user").Base(err)
		}
		account := new(VMessAccount)
		if err := json.Unmarshal(rawData, account); err != nil {
			return nil, newError("invalid VMess user").Base(err)
		}
		user.Account = serial.ToTypedMessage(account.Build())
		config.User[idx] = user
	}

	return config, nil
}

type VMessOutboundConfig struct {
	Address *cfgcommon.Address `json:"address"`
	Port    uint16             `json:"port"`
	Users   []json.RawMessage  `json:"users"`
}

// Build implements Buildable
func (c *VMessOutboundConfig) Build() (proto.Message, error) {
	if len(c.Users) == 0 {
		return nil, newError("0 user configured for VMess outbound")
	}
	if c.Address == nil {
		return nil, newError("address is not set in VMess outbound config")
	}
	spec := &protocol.ServerEndpoint{
		Address: c.Address.Build(),
		Port:    uint32(c.Port),
	}
	for _, rawUser := range c.Users {
		user := new(protocol.User)
		if err := json.Unmarshal(rawUser, user); err != nil {
			return nil, newError("invalid VMess user").Base(err)
		}
		account := new(VMessAccount)
		if err := json.Unmarshal(rawUser, account); err != nil {
			return nil, newError("invalid VMess user").Base(err)
		}
		user.Account = serial.ToTypedMessage(account.Build())
		spec.User = append(spec.User, user)
	}
	return &outbound.Config{
		Server: spec,
	}, nil
}
