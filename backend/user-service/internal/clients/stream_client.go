package clients

import (
	"context"
	"fmt"

	pb "github.com/Josy-coder/stream-service/proto/stream/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StreamServiceClient struct {
	client pb.StreamServiceClient
	conn   *grpc.ClientConn
}

func NewStreamServiceClient(address string) (*StreamServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stream service: %w", err)
	}

	client := pb.NewStreamServiceClient(conn)
	return &StreamServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *StreamServiceClient) Close() error {
	return c.conn.Close()
}

// GetUserStreams gets all streams for a user
func (c *StreamServiceClient) GetUserStreams(ctx context.Context, userID int32) ([]*pb.Stream, error) {
	resp, err := c.client.ListStreams(ctx, &pb.ListStreamsRequest{
		UserId: &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user streams: %w", err)
	}
	return resp.Streams, nil
}

// DeleteUserStreams deletes all streams for a user
func (c *StreamServiceClient) DeleteUserStreams(ctx context.Context, userID int32) error {
	_, err := c.client.DeleteUserStreams(ctx, &pb.DeleteUserStreamsRequest{
		UserId: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user streams: %w", err)
	}
	return nil
}
