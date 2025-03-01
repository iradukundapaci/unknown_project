package grpcclient

import (
	"context"

	"github.com/clementus360/stream-service/proto"
	"github.com/clementus360/stream-service/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Define the streamServiceServer struct
type StreamServiceServer struct {
	proto.UnimplementedStreamServiceServer
	GrpcClient Client
}

// Implement the CreateStream method for gRPC
func (s *StreamServiceServer) CreateStream(ctx context.Context, req *proto.CreateStreamRequest) (*proto.StreamResponse, error) {
	logger := logrus.New()

	// Generate stream key only if not provided
	if req.StreamKey == "" {
		req.StreamKey = utils.GenerateStreamKey()
	} else {
		// Optionally validate the stream key format
		if len(req.StreamKey) < 10 {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid stream key")
		}
	}

	// Call gRPC to create the stream
	streamResponse, err := s.GrpcClient.Client.CreateStream(ctx, req)
	if err != nil {
		logger.Errorf("Failed to create stream via gRPC: %v", err)
		return nil, err
	}

	return streamResponse, nil
}

// Implement the DeleteStream method for gRPC
func (s *StreamServiceServer) DeleteStream(ctx context.Context, req *proto.DeleteStreamRequest) (*emptypb.Empty, error) {
	logger := logrus.New()

	// Call gRPC to delete the stream
	_, err := s.GrpcClient.Client.DeleteStream(ctx, req)
	if err != nil {
		logger.Errorf("Failed to delete stream via gRPC: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Implement the ListStreams method for gRPC
func (s *StreamServiceServer) ListStreams(ctx context.Context, req *proto.ListStreamsRequest) (*proto.ListStreamsResponse, error) {
	logger := logrus.New()

	// Call gRPC to list streams
	streamResponse, err := s.GrpcClient.Client.ListStreams(ctx, req)
	if err != nil {
		logger.Errorf("Failed to list streams via gRPC: %v", err)
		return nil, err
	}

	return streamResponse, nil
}

// Implement the GetStream method for gRPC
func (s *StreamServiceServer) GetStream(ctx context.Context, req *proto.GetStreamRequest) (*proto.StreamResponse, error) {
	logger := logrus.New()

	// Call gRPC to get the stream info
	streamResponse, err := s.GrpcClient.Client.GetStream(ctx, req)
	if err != nil {
		logger.Errorf("Failed to get stream info via gRPC: %v", err)
		return nil, err
	}

	return streamResponse, nil
}

// Implement the UpdateStream method for gRPC
func (s *StreamServiceServer) UpdateStream(ctx context.Context, req *proto.UpdateStreamRequest) (*proto.StreamResponse, error) {
	logger := logrus.New()

	// Call gRPC to update the stream info
	streamResponse, err := s.GrpcClient.Client.UpdateStream(ctx, req)

	if err != nil {
		logger.Errorf("Failed to update stream info via gRPC: %v", err)
		return nil, err
	}

	return streamResponse, nil
}
