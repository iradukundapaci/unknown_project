package service

import (
	"context"
	"errors"
	"github.com/Josy-coder/comment-service/internal/clients"
	"time"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrCommentTooLong  = errors.New("comment too long")
	ErrEmptyComment    = errors.New("empty comment")
	ErrUserNotFound    = errors.New("user not found")
	ErrStreamNotFound  = errors.New("stream not found")
	ErrStreamCompleted = errors.New("cannot comment on completed stream")
	ErrInvalidPage     = errors.New("invalid page number")
	ErrInvalidPageSize = errors.New("invalid page size")
)

const (
	MaxCommentLength = 1000
	DefaultPageSize  = 10
	MaxPageSize      = 100
)

type Comment struct {
	ID        int32
	Content   string
	UserID    int32
	StreamID  int32
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type CommentFilter struct {
	UserID   *int32
	StreamID *int32
	Page     int32
	PageSize int32
}

type CommentService struct {
	dbClient     *clients.DBServiceClient
	userClient   *clients.UserServiceClient
	streamClient *clients.StreamServiceClient
}

func NewCommentService(dbClient *clients.DBServiceClient, userClient *clients.UserServiceClient, streamClient *clients.StreamServiceClient) *CommentService {
	return &CommentService{
		dbClient:     dbClient,
		userClient:   userClient,
		streamClient: streamClient,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, content string, userID, streamID int32) (*Comment, error) {
	if err := validateContent(content); err != nil {
		return nil, err
	}

	// Validate user exists
	if _, err := s.userClient.GetUser(ctx, userID); err != nil {
		return nil, ErrUserNotFound
	}

	// Validate stream exists and is active
	stream, err := s.streamClient.GetStream(ctx, streamID)
	if err != nil {
		return nil, ErrStreamNotFound
	}

	if stream.Status == "COMPLETE" {
		return nil, ErrStreamCompleted
	}

	comment := &Comment{
		Content:   content,
		UserID:    userID,
		StreamID:  streamID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.dbClient.CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) GetComment(ctx context.Context, id int32) (*Comment, error) {
	comment, err := s.dbClient.GetComment(ctx, id)
	if err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) UpdateComment(ctx context.Context, id int32, content string) (*Comment, error) {
	if err := validateContent(content); err != nil {
		return nil, err
	}

	comment, err := s.dbClient.GetComment(ctx, id)
	if err != nil {
		return nil, err
	}

	comment.Content = content
	comment.UpdatedAt = time.Now()

	if err := s.dbClient.UpdateComment(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, id int32) error {
	return s.dbClient.DeleteComment(ctx, id)
}

func (s *CommentService) ListComments(ctx context.Context, filter CommentFilter) ([]*Comment, int32, error) {
	if err := validateFilter(&filter); err != nil {
		return nil, 0, err
	}

	return s.dbClient.ListComments(ctx, &filter)
}

func validateContent(content string) error {
	if len(content) == 0 {
		return ErrEmptyComment
	}
	if len(content) > MaxCommentLength {
		return ErrCommentTooLong
	}
	return nil
}

func validateFilter(filter *CommentFilter) error {
	if filter.Page < 1 {
		return ErrInvalidPage
	}

	if filter.PageSize < 1 {
		filter.PageSize = DefaultPageSize
	} else if filter.PageSize > MaxPageSize {
		filter.PageSize = MaxPageSize
	}

	return nil
}
