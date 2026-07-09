package bootstrap

import (
	"github.com/vsayfb/gig-platform-core-service/config"
	"github.com/vsayfb/gig-platform-core-service/pkg/grpcserver"
	pb "github.com/vsayfb/gig-platform-protos/contracts"
)

func newGRPCServer(cfg *config.Config, svcs *services) *grpcserver.Server {
	grpcHandler := grpcserver.NewGRPCHandler(svcs.user)

	grpcService := grpcserver.New(cfg.GRPC.Port)

	pb.RegisterUserServiceServer(grpcService.GRPCServer(), grpcHandler)

	return grpcService
}
