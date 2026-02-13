package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/PatiharnKam/AiLaw/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type authService struct {
	cfg     *config.Config
	storage AuthStorage
}

func NewService(cfg *config.Config, storage AuthStorage) *authService {
	return &authService{
		cfg:     cfg,
		storage: storage,
	}
}

func (s *authService) GetGoogleLoginURL(email string) string {
	baseURL := "https://accounts.google.com/o/oauth2/v2/auth"

	params := url.Values{}
	params.Add("client_id", s.cfg.Google.ClientID)
	params.Add("redirect_uri", s.cfg.Google.RedirectURI)
	params.Add("response_type", "code")
	params.Add("scope", "openid email profile")
	params.Add("access_type", "offline")
	params.Add("prompt", "consent")

	if email != "" {
		params.Add("login_hint", email)
	}

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (s *authService) HandleGoogleCallback(ctx context.Context, req GoogleCallbackRequest) (*LoginResponse, error) {
	googleToken, err := s.exchangeGoogleCode(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %v", err)
	}

	userInfo, err := s.getGoogleUserInfo(googleToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}

	user := User{
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}

	now := time.Now()
	resp, err := s.storage.CheckUserByEmail(ctx, userInfo.Email)
	if resp == nil {
		user.ID = uuid.NewString()
		user.CreatedAt = now
		user.UpdatedAt = now

		err := s.storage.CreateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}
	} else {
		user.ID = resp.ID
		user.UpdatedAt = now

		err := s.storage.UpdateUser(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("failed to update user: %v", err)
		}
	}

	tokenPair, err := s.generateTokenPair(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	response := LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		UserId:       user.ID,
	}

	return &response, nil
}

func (s *authService) exchangeGoogleCode(code string) (string, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", s.cfg.Google.ClientID)
	data.Set("client_secret", s.cfg.Google.ClientSecret)
	data.Set("redirect_uri", s.cfg.Google.RedirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("google token error: %s", body)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.AccessToken, nil
}

func (s *authService) getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"

	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo error: %s", body)
	}
	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (s *authService) RefreshTokenService(ctx context.Context, refreshToken string) (*RefreshTokenProcessResponse, error) {
	claims, err := s.validateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	exists, err := s.storage.ValidateRefreshToken(ctx, claims.UserID, refreshToken)
	if err != nil || !exists {
		return nil, fmt.Errorf("refresh token not found or invalid")
	}

	err = s.storage.DeleteRefreshTokens(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old refresh token: %v", err)
	}

	newTokens, err := s.generateTokenPair(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %v", err)
	}

	response := RefreshTokenProcessResponse{
		AccessToken:  newTokens.AccessToken,
		RefreshToken: newTokens.RefreshToken,
		UserId:       claims.UserID,
	}

	return &response, nil
}

func (s *authService) validateRefreshToken(refreshTokenString string) (*JWTClaims, error) {
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.cfg.JWT.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	token, err := jwt.ParseWithClaims(refreshTokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired refresh token: %v", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (s *authService) LogoutProcess(ctx context.Context, refreshToken string) error {
	err := s.storage.DeleteRefreshTokens(ctx, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to delete old refresh token: %v", err)
	}
	return nil
}
