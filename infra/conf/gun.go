package conf

import (
	"github.com/golang/protobuf/proto"

	"github.com/Shadowsocks-NET/v2ray-go/v4/transport/internet/grpc"
)

type GunConfig struct {
	ServiceName string `json:"serviceName"`
}

func (g GunConfig) Build() (proto.Message, error) {
	return &grpc.Config{ServiceName: g.ServiceName}, nil
}
