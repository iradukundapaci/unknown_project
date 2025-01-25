package clients

import (
	"context"
	pb "github.com/Josy-coder/stream-service/proto/stream/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StreamServiceClient struct {
	client pb.StreamServiceClient
}

func NewStreamServiceClient(address string) (*StreamServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &StreamServiceClient{
		client: pb.NewStreamServiceClient(conn),
	}, nil
}

func (c *StreamServiceClient) GetStream(ctx context.Context, streamID int32) (*pb.StreamResponse, error) {
	return c.client.GetStream(ctx, &pb.GetStreamRequest{
		Id: streamID,
	})
}
