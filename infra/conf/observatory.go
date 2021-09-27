package conf

import (
	"github.com/golang/protobuf/proto"

	"github.com/Shadowsocks-NET/v2ray-go/v4/app/observatory"
	"github.com/Shadowsocks-NET/v2ray-go/v4/infra/conf/cfgcommon/duration"
)

type ObservatoryConfig struct {
	SubjectSelector []string          `json:"subjectSelector"`
	ProbeURL        string            `json:"probeURL"`
	ProbeInterval   duration.Duration `json:"probeInterval"`
}

func (o *ObservatoryConfig) Build() (proto.Message, error) {
	return &observatory.Config{SubjectSelector: o.SubjectSelector, ProbeUrl: o.ProbeURL, ProbeInterval: int64(o.ProbeInterval)}, nil
}
