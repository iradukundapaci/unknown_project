package ports

import (
	"context"
	"errors"

	"github.com/Josy-coder/user-service/internal/domain"
	"github.com/Josy-coder/user-service/internal/service"
	pb "github.com/Josy-coder/user-service/proto/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	pb.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewGRPCServer(svc *service.UserService) pb.UserServiceServer {
	return &GRPCServer{
		svc: svc,
	}
}

func (s *GRPCServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	user := &domain.User{
		ClerkID:         req.ClerkId,
		Email:           req.Email,
		Username:        req.Username,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		ProfileImageURL: req.ProfileImageUrl,
	}

	createdUser, err := s.svc.CreateUser(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEmailTaken):
			return nil, status.Error(codes.AlreadyExists, "email already taken")
		case errors.Is(err, domain.ErrUsernameTaken):
			return nil, status.Error(codes.AlreadyExists, "username already taken")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.UserResponse{
		User: toProtoUser(createdUser),
	}, nil
}

func (s *GRPCServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	user, err := s.svc.GetUser(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *GRPCServer) GetUserByClerkID(ctx context.Context, req *pb.GetUserByClerkIDRequest) (*pb.UserResponse, error) {
	user, err := s.svc.GetUserByClerkID(ctx, req.ClerkId)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UserResponse{
		User: toProtoUser(user),
	}, nil
}

func (s *GRPCServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	user := &domain.User{
		ID:              req.Id,
		Username:        req.Username,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		ProfileImageURL: req.ProfileImageUrl,
	}

	updatedUser, err := s.svc.UpdateUser(ctx, user)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserNotFound):
			return nil, status.Error(codes.NotFound, "user not found")
		case errors.Is(err, domain.ErrUsernameTaken):
			return nil, status.Error(codes.AlreadyExists, "username already taken")
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.UserResponse{
		User: toProtoUser(updatedUser),
	}, nil
}

func (s *GRPCServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	err := s.svc.DeleteUser(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	filter := domain.UserFilter{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	users, total, err := s.svc.ListUsers(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoUsers := make([]*pb.User, len(users))
	for i, user := range users {
		protoUsers[i] = toProtoUser(user)
	}

	return &pb.ListUsersResponse{
		Users:      protoUsers,
		TotalCount: total,
	}, nil
}

func (s *GRPCServer) SyncUserWithClerk(ctx context.Context, req *pb.SyncUserWithClerkRequest) (*pb.UserResponse, error) {
	user, err := s.svc.SyncUserWithClerk(ctx, req.ClerkId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.UserResponse{
		User: toProtoUser(user),
	}, nil
}

func toProtoUser(u *domain.User) *pb.User {
	if u == nil {
		return nil
	}

	return &pb.User{
		Id:              u.ID,
		ClerkId:         u.ClerkID,
		Email:           u.Email,
		Username:        u.Username,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		ProfileImageUrl: u.ProfileImageURL,
		CreatedAt:       timestamppb.New(u.CreatedAt),
		UpdatedAt:       timestamppb.New(u.UpdatedAt),
		LastLogin:       timestamppb.New(u.LastLogin),
	}
}
