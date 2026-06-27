package grpcserver

import (
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

type Server struct {
	server *grpc.Server
	port   string
}

func New(port string, opts ...grpc.ServerOption) *Server {
	return &Server{
		server: grpc.NewServer(opts...),
		port:   port,
	}
}

func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", ":"+s.port)

	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	return s.server.Serve(lis)
}

func (s *Server) Stop() {
	slog.Info("stopping gRPC server")

	s.server.GracefulStop()
}
