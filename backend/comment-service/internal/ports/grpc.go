package ports

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/Josy-coder/comment-service/internal/domain"
	"github.com/Josy-coder/comment-service/internal/service"
	pb "github.com/Josy-coder/comment-service/proto/comment/v1"
)

type GRPCServer struct {
	pb.UnimplementedCommentServiceServer
	svc service.Service
}

func NewGRPCServer(repo domain.CommentRepository) pb.CommentServiceServer {
	return &GRPCServer{
		svc: service.NewCommentService(repo),
	}
}

func (s *GRPCServer) CreateComment(_ context.Context, req *pb.CreateCommentRequest) (*pb.CommentResponse, error) {
	comment, err := s.svc.CreateComment(req.Content, req.UserId, req.StreamId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CommentResponse{
		Comment: toProtoComment(comment),
	}, nil
}

func (s *GRPCServer) GetComment(_ context.Context, req *pb.GetCommentRequest) (*pb.CommentResponse, error) {
	comment, err := s.svc.GetComment(req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrCommentNotFound) {
			return nil, status.Error(codes.NotFound, "comment not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CommentResponse{
		Comment: toProtoComment(comment),
	}, nil
}

func (s *GRPCServer) UpdateComment(_ context.Context, req *pb.UpdateCommentRequest) (*pb.CommentResponse, error) {
	comment, err := s.svc.UpdateComment(req.Id, req.Content)
	if err != nil {
		if errors.Is(err, domain.ErrCommentNotFound) {
			return nil, status.Error(codes.NotFound, "comment not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.CommentResponse{
		Comment: toProtoComment(comment),
	}, nil
}

func (s *GRPCServer) DeleteComment(_ context.Context, req *pb.DeleteCommentRequest) (*emptypb.Empty, error) {
	err := s.svc.DeleteComment(req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrCommentNotFound) {
			return nil, status.Error(codes.NotFound, "comment not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) ListComments(_ context.Context, req *pb.ListCommentsRequest) (*pb.ListCommentsResponse, error) {
	filter := domain.CommentFilter{
		UserID:   req.UserId,
		StreamID: req.StreamId,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}

	comments, total, err := s.svc.ListComments(filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	protoComments := make([]*pb.Comment, len(comments))
	for i, comment := range comments {
		protoComments[i] = toProtoComment(comment)
	}

	return &pb.ListCommentsResponse{
		Comments:   protoComments,
		TotalCount: total,
	}, nil
}

func toProtoComment(c *domain.Comment) *pb.Comment {
	if c == nil {
		return nil
	}

	return &pb.Comment{
		Id:        c.ID,
		Content:   c.Content,
		UserId:    c.UserID,
		StreamId:  c.StreamID,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}
