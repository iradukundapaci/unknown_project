package domain

import (
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

func ValidateComment(content string) error {
	if len(content) == 0 {
		return ErrEmptyComment
	}

	if len(content) > MaxCommentLength {
		return ErrCommentTooLong
	}

	return nil
}
