package extension

import (
	"context"

	"github.com/Shadowsocks-NET/v2ray-go/v4/features"
	"github.com/golang/protobuf/proto"
)

type Observatory interface {
	features.Feature

	GetObservation(ctx context.Context) (proto.Message, error)
}

func ObservatoryType() interface{} {
	return (*Observatory)(nil)
}
