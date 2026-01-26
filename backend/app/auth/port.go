package auth

import (
	"context"
	"time"
)

type AuthService interface {
	GetGoogleLoginURL(email string) string
	HandleGoogleCallback(ctx context.Context, req GoogleCallbackRequest) (*LoginResponse, error)
	RefreshTokenService(ctx context.Context, refreshToken string) (*RefreshTokenProcessResponse, error)
	LogoutProcess(ctx context.Context, refreshToken string) error
}

type AuthStorage interface {
	CheckUserByEmail(ctx context.Context, email string) (*User, error)
	CreateUser(ctx context.Context, user User) error
	UpdateUser(ctx context.Context, user User) error

	StoreRefreshToken(ctx context.Context, userID, token string, expiredAt time.Time) error
	ValidateRefreshToken(ctx context.Context, userID, token string) (bool, error)
	DeleteRefreshTokens(ctx context.Context, refreshToken string) error
}

type GoogleCallbackRequest struct {
	Code string `form:"code" binding:"required"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

type User struct {
	ID        string
	Email     string
	Name      string
	Picture   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserId       string `json:"userId"`
}

type RefreshTokenProcessResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserId       string `json:"userId"`
}
