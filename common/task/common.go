package task

import "github.com/Shadowsocks-NET/v2ray-go/v4/common"

// Close returns a func() that closes v.
func Close(v interface{}) func() error {
	return func() error {
		return common.Close(v)
	}
}
