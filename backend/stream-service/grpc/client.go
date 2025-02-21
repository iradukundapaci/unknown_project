package grpcclient

import (
	"context"
	"fmt"
	"os"

	"github.com/clementus360/stream-service/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	Conn   *grpc.ClientConn
	Client proto.StreamServiceClient
}

func NewClient(ctx context.Context) (*Client, error) {
	logger := logrus.New()

	// Get the database service address from environment variable
	dbAddress := os.Getenv("DB_SERVICE_ADDRESS")
	if dbAddress == "" {
		dbAddress = "localhost:8080" // Default address
	}

	// Create a connection to the server
	conn, err := grpc.NewClient(
		dbAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.WithError(err).Error("Failed to connect to database service")
		return nil, fmt.Errorf("failed to connect to the database service: %v", err)
	}

	// Ensure graceful shutdown by deferring closing the connection
	client := proto.NewStreamServiceClient(conn)

	fmt.Println("Connected to database service at", dbAddress)

	return &Client{
		Conn:   conn,
		Client: client,
	}, nil
}

// Graceful shutdown to close the gRPC connection properly
func (c *Client) Close() {
	if err := c.Conn.Close(); err != nil {
		logrus.WithError(err).Error("Failed to close gRPC connection")
	}
}
