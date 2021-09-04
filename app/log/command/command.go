//go:build !confonly
// +build !confonly

package command

import (
	"context"

	core "github.com/Shadowsocks-NET/v2ray-go/v4"
	"github.com/Shadowsocks-NET/v2ray-go/v4/app/log"
	"github.com/Shadowsocks-NET/v2ray-go/v4/common"
	grpc "google.golang.org/grpc"
)

type LoggerServer struct {
	V *core.Instance
}

// RestartLogger implements LoggerService.
func (s *LoggerServer) RestartLogger(ctx context.Context, request *RestartLoggerRequest) (*RestartLoggerResponse, error) {
	logger := s.V.GetFeature((*log.Instance)(nil))
	if logger == nil {
		return nil, newError("unable to get logger instance")
	}
	if err := logger.Close(); err != nil {
		return nil, newError("failed to close logger").Base(err)
	}
	if err := logger.Start(); err != nil {
		return nil, newError("failed to start logger").Base(err)
	}
	return &RestartLoggerResponse{}, nil
}

func (s *LoggerServer) mustEmbedUnimplementedLoggerServiceServer() {}

type service struct {
	v *core.Instance
}

func (s *service) Register(server *grpc.Server) {
	RegisterLoggerServiceServer(server, &LoggerServer{
		V: s.v,
	})
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		s := core.MustFromContext(ctx)
		return &service{v: s}, nil
	}))
}
