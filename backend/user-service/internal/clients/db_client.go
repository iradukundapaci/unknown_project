package clients

import (
	"context"
	"fmt"

	pb "github.com/Josy-coder/db-service/proto/db/v1"
	"github.com/Josy-coder/user-service/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DBServiceClient struct {
	client pb.DatabaseServiceClient
	conn   *grpc.ClientConn
}

func NewDBServiceClient(address string) (*DBServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database service: %w", err)
	}

	client := pb.NewDatabaseServiceClient(conn)
	return &DBServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *DBServiceClient) Close() error {
	return c.conn.Close()
}

func (c *DBServiceClient) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := c.client.CreateUser(ctx, &pb.CreateUserRequest{
		User: toProtoUser(user),
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (c *DBServiceClient) GetUser(ctx context.Context, id int32) (*domain.User, error) {
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{
		Id: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return toDomainUser(resp.User), nil
}

func (c *DBServiceClient) GetUserByClerkID(ctx context.Context, clerkID string) (*domain.User, error) {
	resp, err := c.client.GetUserByClerkID(ctx, &pb.GetUserByClerkIDRequest{
		ClerkId: clerkID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by clerk ID: %w", err)
	}
	return toDomainUser(resp.User), nil
}

func (c *DBServiceClient) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	resp, err := c.client.GetUserByEmail(ctx, &pb.GetUserByEmailRequest{
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return toDomainUser(resp.User), nil
}

func (c *DBServiceClient) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	resp, err := c.client.GetUserByUsername(ctx, &pb.GetUserByUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return toDomainUser(resp.User), nil
}

func (c *DBServiceClient) UpdateUser(ctx context.Context, user *domain.User) error {
	_, err := c.client.UpdateUser(ctx, &pb.UpdateUserRequest{
		User: toProtoUser(user),
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (c *DBServiceClient) DeleteUser(ctx context.Context, id int32) error {
	_, err := c.client.DeleteUser(ctx, &pb.DeleteUserRequest{
		Id: id,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (c *DBServiceClient) ListUsers(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int32, error) {
	resp, err := c.client.ListUsers(ctx, &pb.ListUsersRequest{
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*domain.User, len(resp.Users))
	for i, user := range resp.Users {
		users[i] = toDomainUser(user)
	}

	return users, resp.TotalCount, nil
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

func toDomainUser(u *pb.User) *domain.User {
	if u == nil {
		return nil
	}

	user := &domain.User{
		ID:              u.Id,
		ClerkID:         u.ClerkId,
		Email:           u.Email,
		Username:        u.Username,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		ProfileImageURL: u.ProfileImageUrl,
	}

	if u.CreatedAt != nil {
		user.CreatedAt = u.CreatedAt.AsTime()
	}
	if u.UpdatedAt != nil {
		user.UpdatedAt = u.UpdatedAt.AsTime()
	}
	if u.LastLogin != nil {
		user.LastLogin = u.LastLogin.AsTime()
	}

	return user
}
