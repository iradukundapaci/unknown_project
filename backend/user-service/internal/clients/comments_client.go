package clients

import (
	"context"
	"fmt"

	pb "github.com/Josy-coder/comment-service/proto/comment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CommentServiceClient struct {
	client pb.CommentServiceClient
	conn   *grpc.ClientConn
}

func NewCommentServiceClient(address string) (*CommentServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to comment service: %w", err)
	}

	client := pb.NewCommentServiceClient(conn)
	return &CommentServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *CommentServiceClient) Close() error {
	return c.conn.Close()
}

// GetUserComments gets all comments for a user
func (c *CommentServiceClient) GetUserComments(ctx context.Context, userID int32) ([]*pb.Comment, error) {
	resp, err := c.client.ListComments(ctx, &pb.ListCommentsRequest{
		UserId: &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user comments: %w", err)
	}
	return resp.Comments, nil
}

// DeleteUserComments deletes all comments for a user
func (c *CommentServiceClient) DeleteUserComments(ctx context.Context, userID int32) error {
	comments, err := c.GetUserComments(ctx, userID)
	if err != nil {
		return err
	}

	for _, comment := range comments {
		_, err := c.client.DeleteComment(ctx, &pb.DeleteCommentRequest{
			Id: comment.Id,
		})
		if err != nil {
			return fmt.Errorf("failed to delete comment %d: %w", comment.Id, err)
		}
	}
	return nil
}
