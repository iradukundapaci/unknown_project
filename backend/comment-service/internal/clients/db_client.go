package clients

import (
	"context"
	"fmt"

	"github.com/Josy-coder/comment-service/internal/domain"
	pb "github.com/Josy-coder/db-service/proto/db/v1"
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

func (c *DBServiceClient) CreateComment(ctx context.Context, comment *domain.Comment) error {
	_, err := c.client.CreateComment(ctx, &pb.CreateCommentRequest{
		Comment: toProtoComment(comment),
	})
	return err
}

func (c *DBServiceClient) GetComment(ctx context.Context, id int32) (*domain.Comment, error) {
	resp, err := c.client.GetComment(ctx, &pb.GetCommentRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return toDomainComment(resp.Comment), nil
}

func (c *DBServiceClient) UpdateComment(ctx context.Context, comment *domain.Comment) error {
	_, err := c.client.UpdateComment(ctx, &pb.UpdateCommentRequest{
		Comment: toProtoComment(comment),
	})
	return err
}

func (c *DBServiceClient) DeleteComment(ctx context.Context, id int32) error {
	_, err := c.client.DeleteComment(ctx, &pb.DeleteCommentRequest{
		Id: id,
	})
	return err
}

func (c *DBServiceClient) ListComments(ctx context.Context, filter domain.CommentFilter) ([]*domain.Comment, int32, error) {
	resp, err := c.client.ListComments(ctx, &pb.ListCommentsRequest{
		StreamId: filter.StreamID,
		UserId:   filter.UserID,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	comments := make([]*domain.Comment, len(resp.Comments))
	for i, c := range resp.Comments {
		comments[i] = toDomainComment(c)
	}

	return comments, resp.TotalCount, nil
}

func toProtoComment(c *domain.Comment) *pb.Comment {
	return &pb.Comment{
		Id:        c.ID,
		Content:   c.Content,
		UserId:    c.UserID,
		StreamId:  c.StreamID,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func toDomainComment(c *pb.Comment) *domain.Comment {
	if c == nil {
		return nil
	}

	comment := &domain.Comment{
		ID:       c.Id,
		Content:  c.Content,
		UserID:   c.UserId,
		StreamID: c.StreamId,
	}

	if c.CreatedAt != nil {
		comment.CreatedAt = c.CreatedAt.AsTime()
	}
	if c.UpdatedAt != nil {
		comment.UpdatedAt = c.UpdatedAt.AsTime()
	}
	if c.DeletedAt != nil {
		deletedAt := c.DeletedAt.AsTime()
		comment.DeletedAt = &deletedAt
	}

	return comment
}
