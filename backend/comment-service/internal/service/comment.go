package service

import (
	"context"
	"errors"
	"time"

	"github.com/Josy-coder/comment-service/internal/domain"
)

var (
	ErrInvalidPage     = errors.New("invalid page number")
	ErrInvalidPageSize = errors.New("invalid page size")
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type DBClient interface {
	CreateComment(ctx context.Context, comment *domain.Comment) error
	GetComment(ctx context.Context, id int32) (*domain.Comment, error)
	UpdateComment(ctx context.Context, comment *domain.Comment) error
	DeleteComment(ctx context.Context, id int32) error
	ListComments(ctx context.Context, filter domain.CommentFilter) ([]*domain.Comment, int32, error)
}

type UserClient interface {
	GetUser(ctx context.Context, userID int32) error
}

type StreamClient interface {
	GetStream(ctx context.Context, streamID int32) error
}

type CommentService struct {
	dbClient     DBClient
	userClient   UserClient
	streamClient StreamClient
}

func NewCommentService(dbClient DBClient, userClient UserClient, streamClient StreamClient) *CommentService {
	return &CommentService{
		dbClient:     dbClient,
		userClient:   userClient,
		streamClient: streamClient,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, content string, userID, streamID int32) (*domain.Comment, error) {
	if err := domain.ValidateCommentContent(content); err != nil {
		return nil, err
	}

	// Validate user exists
	if err := s.userClient.GetUser(ctx, userID); err != nil {
		return nil, err
	}

	// Validate stream exists
	if err := s.streamClient.GetStream(ctx, streamID); err != nil {
		return nil, err
	}

	comment := &domain.Comment{
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

func (s *CommentService) GetComment(ctx context.Context, id int32) (*domain.Comment, error) {
	return s.dbClient.GetComment(ctx, id)
}

func (s *CommentService) UpdateComment(ctx context.Context, id int32, content string) (*domain.Comment, error) {
	if err := domain.ValidateCommentContent(content); err != nil {
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

func (s *CommentService) ListComments(ctx context.Context, filter domain.CommentFilter) ([]*domain.Comment, int32, error) {
	if err := validateFilter(&filter); err != nil {
		return nil, 0, err
	}

	return s.dbClient.ListComments(ctx, filter)
}

func validateFilter(filter *domain.CommentFilter) error {
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
