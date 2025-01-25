package clients

import (
	"context"
	pb "github.com/Josy-coder/user-service/proto/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserServiceClient struct {
	client pb.UserServiceClient
}

func NewUserServiceClient(address string) (*UserServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &UserServiceClient{
		client: pb.NewUserServiceClient(conn),
	}, nil
}

func (c *UserServiceClient) GetUser(ctx context.Context, userID int32) (*pb.UserResponse, error) {
	return c.client.GetUser(ctx, &pb.GetUserRequest{
		Id: userID,
	})
}
