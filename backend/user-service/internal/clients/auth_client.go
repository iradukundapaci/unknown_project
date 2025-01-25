package clients

import (
	"context"
	"errors"
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"strings"

	"github.com/clerkinc/clerk-sdk-go"
	"google.golang.org/grpc/metadata"
)

var (
	ErrNoToken          = errors.New("no auth token provided")
	ErrInvalidToken     = errors.New("invalid auth token")
	ErrTokenExpired     = errors.New("auth token expired")
	ErrNotAuthenticated = errors.New("user not authenticated")
)

type AuthClient struct {
	clerkClient clerk.Client
}

func NewAuthClient(clerkClient clerk.Client) *AuthClient {
	return &AuthClient{
		clerkClient: clerkClient,
	}
}

// GetUserFromContext extracts and validates the auth token from the context,
// then returns the associated Clerk user
func (c *AuthClient) GetUserFromContext(ctx context.Context) (*clerk.User, error) {
	token, err := c.extractToken(ctx)
	if err != nil {
		return nil, err
	}

	// Verify the session token
	claims, err := c.clerkClient.VerifyToken(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get the user from Clerk
	user, err := c.clerkClient.Users().Read(claims.Subject)
	if err != nil {
		return nil, ErrNotAuthenticated
	}

	return user, nil
}

// extractToken extracts the Bearer token from the gRPC metadata
func (c *AuthClient) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrNoToken
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", ErrNoToken
	}

	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", ErrInvalidToken
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", ErrInvalidToken
	}

	return token, nil
}

// ValidateToken validates the provided token
func (c *AuthClient) ValidateToken(token string) error {
	_, err := c.clerkClient.VerifyToken(token)
	if err != nil {
		return ErrInvalidToken
	}
	return nil
}
