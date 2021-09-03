package signal_test

import (
	"testing"

	. "github.com/Shadowsocks-NET/v2ray-go/v4/common/signal"
)

func TestNotifierSignal(t *testing.T) {
	n := NewNotifier()

	w := n.Wait()
	n.Signal()

	select {
	case <-w:
	default:
		t.Fail()
	}
}
