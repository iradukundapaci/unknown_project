package service

import (
	"context"
	"errors"
	"time"

	"github.com/Josy-coder/user-service/internal/domain"
	"github.com/clerkinc/clerk-sdk-go"
)

var (
	ErrInvalidPage     = errors.New("invalid page number")
	ErrInvalidPageSize = errors.New("invalid page size")
)

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type UserService struct {
	repo        domain.UserRepository
	clerkClient clerk.Client
}

func NewUserService(repo domain.UserRepository, clerkClient clerk.Client) *UserService {
	return &UserService{
		repo:        repo,
		clerkClient: clerkClient,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	if err := domain.ValidateNewUser(user); err != nil {
		return nil, err
	}

	// Check if user with same email exists
	if existingUser, _ := s.repo.GetUserByEmail(ctx, user.Email); existingUser != nil {
		return nil, domain.ErrEmailTaken
	}

	// Check if username is taken
	if existingUser, _ := s.repo.GetUserByUsername(ctx, user.Username); existingUser != nil {
		return nil, domain.ErrUsernameTaken
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.LastLogin = now

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id int32) (*domain.User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *UserService) GetUserByClerkID(ctx context.Context, clerkID string) (*domain.User, error) {
	return s.repo.GetUserByClerkID(ctx, clerkID)
}

func (s *UserService) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	existingUser, err := s.repo.GetUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Check if new username is taken
	if user.Username != existingUser.Username {
		if existing, _ := s.repo.GetUserByUsername(ctx, user.Username); existing != nil {
			return nil, domain.ErrUsernameTaken
		}
	}

	user.UpdatedAt = time.Now()
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int32) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *UserService) ListUsers(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int32, error) {
	if err := validateFilter(filter); err != nil {
		return nil, 0, err
	}

	return s.repo.ListUsers(ctx, filter)
}

func (s *UserService) SyncUserWithClerk(ctx context.Context, clerkID string) (*domain.User, error) {
	// Get user from Clerk
	clerkUser, err := s.clerkClient.Users().Read(clerkID)
	if err != nil {
		return nil, err
	}

	// Check if user exists in our database
	user, err := s.repo.GetUserByClerkID(ctx, clerkID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	// If user doesn't exist, create new user
	if user == nil {
		user = &domain.User{
			ClerkID:         clerkID,
			Email:           clerkUser.EmailAddresses[0].EmailAddress,
			Username:        clerkUser.Username,
			FirstName:       clerkUser.FirstName,
			LastName:        clerkUser.LastName,
			ProfileImageURL: clerkUser.ProfileImageURL,
		}
		return s.CreateUser(ctx, user)
	}

	// Update existing user with Clerk data
	user.Email = clerkUser.EmailAddresses[0].EmailAddress
	user.Username = clerkUser.Username
	user.FirstName = clerkUser.FirstName
	user.LastName = clerkUser.LastName
	user.ProfileImageURL = clerkUser.ProfileImageURL
	user.LastLogin = time.Now()

	return s.UpdateUser(ctx, user)
}

func validateFilter(filter domain.UserFilter) error {
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
