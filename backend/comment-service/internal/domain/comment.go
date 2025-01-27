package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrCommentTooLong  = errors.New("comment too long")
	ErrEmptyComment    = errors.New("empty comment")
)

const MaxCommentLength = 1000

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
	PageSize int32
	Page     int32
}

type CommentRepository interface {
	CreateComment(ctx context.Context, comment *Comment) error
	GetComment(ctx context.Context, id int32) (*Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, id int32) error
	ListComments(ctx context.Context, filter CommentFilter) ([]*Comment, int32, error)
}

func ValidateCommentContent(content string) error {
	if len(content) == 0 {
		return ErrEmptyComment
	}

	if len(content) > MaxCommentLength {
		return ErrCommentTooLong
	}

	return nil
}
