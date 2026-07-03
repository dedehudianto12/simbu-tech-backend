package service

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dedehudianto12/simbu-tech-backend/internal/repository"
	jwtpkg "github.com/dedehudianto12/simbu-tech-backend/pkg/jwt"
)

// AuthService handles authentication logic.
type AuthService struct {
	repo        *repository.UserRepo
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo *repository.UserRepo, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  jwtSecret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

var errInvalidCredentials = fmt.Errorf("invalid credentials")

// Login authenticates a user and returns access and refresh tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("auth_service.Login: %w", errInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("auth_service.Login: %w", errInvalidCredentials)
	}

	accessToken, err := jwtpkg.NewAccessToken(user.ID.String(), user.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return "", "", fmt.Errorf("auth_service.Login: %w", err)
	}

	refreshToken, err := jwtpkg.NewRefreshToken(user.ID.String(), user.Role, s.jwtSecret, s.refreshTTL)
	if err != nil {
		return "", "", fmt.Errorf("auth_service.Login: %w", err)
	}

	return accessToken, refreshToken, nil
}

// RefreshToken validates a refresh token and issues a new access token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := jwtpkg.VerifyToken(refreshToken, s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("auth_service.RefreshToken: %w", err)
	}

	accessToken, err := jwtpkg.NewAccessToken(claims.UserID, claims.Role, s.jwtSecret, s.accessTTL)
	if err != nil {
		return "", fmt.Errorf("auth_service.RefreshToken: %w", err)
	}

	return accessToken, nil
}
