package clients

import (
	"context"
	"github.com/Josy-coder/comment-service/internal/service"
	dbpb "github.com/Josy-coder/db-service/proto/db/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DBServiceClient struct {
	client dbpb.DatabaseServiceClient
	conn   *grpc.ClientConn
}

func NewDBServiceClient(address string) (*DBServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &DBServiceClient{
		client: dbpb.NewDatabaseServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *DBServiceClient) Close() error {
	return c.conn.Close()
}

func (c *DBServiceClient) CreateComment(ctx context.Context, comment *service.Comment) error {
	_, err := c.client.CreateComment(ctx, &dbpb.CreateCommentRequest{
		Comment: &dbpb.Comment{
			Content:   comment.Content,
			UserId:    comment.UserID,
			StreamId:  comment.StreamID,
			CreatedAt: timestamppb.New(comment.CreatedAt),
			UpdatedAt: timestamppb.New(comment.UpdatedAt),
		},
	})
	return err
}

func (c *DBServiceClient) GetComment(ctx context.Context, id int32) (*service.Comment, error) {
	resp, err := c.client.GetComment(ctx, &dbpb.GetCommentRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}

	return toServiceComment(resp.Comment), nil
}

func (c *DBServiceClient) UpdateComment(ctx context.Context, comment *service.Comment) error {
	_, err := c.client.UpdateComment(ctx, &dbpb.UpdateCommentRequest{
		Comment: &dbpb.Comment{
			Id:        comment.ID,
			Content:   comment.Content,
			UpdatedAt: timestamppb.New(comment.UpdatedAt),
		},
	})
	return err
}

func (c *DBServiceClient) DeleteComment(ctx context.Context, id int32) error {
	_, err := c.client.DeleteComment(ctx, &dbpb.DeleteCommentRequest{
		Id: id,
	})
	return err
}

func (c *DBServiceClient) ListComments(ctx context.Context, filter *service.CommentFilter) ([]*service.Comment, int32, error) {
	resp, err := c.client.ListComments(ctx, &dbpb.ListCommentsRequest{
		StreamId: filter.StreamID,
		UserId:   filter.UserID,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	comments := make([]*service.Comment, len(resp.Comments))
	for i, c := range resp.Comments {
		comments[i] = toServiceComment(c)
	}

	return comments, resp.TotalCount, nil
}

func toServiceComment(c *dbpb.Comment) *service.Comment {
	if c == nil {
		return nil
	}

	comment := &service.Comment{
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
