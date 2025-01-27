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

func (c *StreamServiceClient) GetStream(ctx context.Context, streamID int32) error {
	_, err := c.client.GetStream(ctx, &pb.GetStreamRequest{
		Id: streamID,
	})
	return err
}
