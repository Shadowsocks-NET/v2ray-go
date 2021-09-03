package mtproto

import (
	"github.com/Shadowsocks-NET/v2ray-go/v4/common/protocol"
)

func (a *Account) Equals(another protocol.Account) bool {
	aa, ok := another.(*Account)
	if !ok {
		return false
	}

	if len(a.Secret) != len(aa.Secret) {
		return false
	}

	for i, v := range a.Secret {
		if v != aa.Secret[i] {
			return false
		}
	}

	return true
}
