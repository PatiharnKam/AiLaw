package auth

import (
	"context"
	"fmt"
	"time"

	// "github.com/PatiharnKam/AiLaw/app/auth"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *authService) generateTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(s.cfg.JWT.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	accessClaims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(privateKey) // ✅ ใช้ privateKey แทน string
	if err != nil {
		return nil, err
	}

	refreshClaims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(privateKey) // ✅
	if err != nil {
		return nil, err
	}

	expiredAt := time.Now().Add(30 * 24 * time.Hour)
	err = s.storage.StoreRefreshToken(ctx, userID, refreshTokenString, expiredAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %v", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}
