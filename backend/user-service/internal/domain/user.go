package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidUsername = errors.New("invalid username")
	ErrClerkIDRequired = errors.New("clerk ID is required")
	ErrUsernameTaken   = errors.New("username already taken")
	ErrEmailTaken      = errors.New("email already taken")
)

type User struct {
	ID              int32
	ClerkID         string
	Email           string
	Username        string
	FirstName       string
	LastName        string
	ProfileImageURL string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastLogin       time.Time
}

type UserFilter struct {
	PageSize int32
	Page     int32
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id int32) (*User, error)
	GetUserByClerkID(ctx context.Context, clerkID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int32) error
	ListUsers(ctx context.Context, filter UserFilter) ([]*User, int32, error)
}

func ValidateNewUser(user *User) error {
	if user.ClerkID == "" {
		return ErrClerkIDRequired
	}

	if err := validateEmail(user.Email); err != nil {
		return err
	}

	if err := validateUsername(user.Username); err != nil {
		return err
	}

	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	// Add more email validation if needed
	return nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 {
		return ErrInvalidUsername
	}
	// Add more username validation if needed
	return nil
}
