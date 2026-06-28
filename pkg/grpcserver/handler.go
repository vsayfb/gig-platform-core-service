package grpcserver

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vsayfb/gig-platform-core-service/internal/user"
	pb "github.com/vsayfb/gig-platform-protos/contracts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedUserServiceServer
	svc *user.UserService
}

func NewGRPCHandler(svc *user.UserService) *GRPCHandler {
	return &GRPCHandler{
		svc: svc,
	}
}

func (h *GRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	id, err := uuid.Parse(req.UserId)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}

	found, err := h.svc.GetByID(ctx, id)

	if err != nil {

		if errors.Is(err, user.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	return &pb.GetUserResponse{
		Id:        req.UserId,
		Name:      found.Name,
		AvatarUrl: found.AvatarURL,
	}, nil
}
