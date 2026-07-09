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
	svc user.UserService
}

func NewGRPCHandler(svc user.UserService) *GRPCHandler {
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
		User: &pb.User{
			Id:        found.ID.String(),
			Name:      found.Name,
			AvatarUrl: found.AvatarURL,
		},
	}, nil
}

func (h *GRPCHandler) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.UserIds))

	for _, id := range req.UserIds {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %s", id)
		}
		ids = append(ids, parsed)
	}

	summaries, err := h.svc.GetSummaries(ctx, ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "internal error")
	}

	resp := &pb.GetUsersResponse{
		Users: make([]*pb.User, 0, len(summaries)),
	}

	for _, s := range summaries {
		resp.Users = append(resp.Users, &pb.User{
			Id:        s.ID.String(),
			Name:      s.Name,
			AvatarUrl: s.AvatarURL,
		})
	}

	return resp, nil
}
