package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/PatiharnKam/AiLaw/app/auth"
	"github.com/PatiharnKam/AiLaw/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func GinJWTMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Error("Authorization header required")
			c.JSON(http.StatusUnauthorized, app.Response{
				Code:    app.UnauthorizedErrorCode,
				Message: app.UnauthorizedErrorMessage,
				Data: JWTErrorActionResponse{
					Action: app.ActionLogout,
				},
			})
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			slog.Error("Invalid authorization header format")
			c.JSON(http.StatusUnauthorized, app.Response{
				Code:    app.UnauthorizedErrorCode,
				Message: app.UnauthorizedErrorMessage,
				Data: JWTErrorActionResponse{
					Action: app.ActionLogout,
				},
			})
			c.Abort()
			return
		}

		// แปลง public key string -> *rsa.PublicKey
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(cfg.JWT.PublicKey))
		if err != nil {
			slog.Error("failed to parse public key")
			c.JSON(http.StatusUnauthorized, app.Response{
				Code:    app.UnauthorizedErrorCode,
				Message: app.UnauthorizedErrorMessage,
				Data: JWTErrorActionResponse{
					Action: app.ActionLogout,
				},
			})
			c.Abort()
			return
		}

		// parse token ด้วย RS256
		token, err := jwt.ParseWithClaims(bearerToken[1], &auth.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// เช็คว่าใช้ algorithm ที่ถูกต้อง
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return pubKey, nil
		})

		// จัดการ Error
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				slog.Error("access token expired go to refresh token : %w", err)
				c.JSON(http.StatusUnauthorized, app.Response{
					Code:    app.UnauthorizedErrorCode,
					Message: app.UnauthorizedErrorMessage,
					Data: JWTErrorActionResponse{
						Action: app.ActionRefresh,
					},
				})
				c.Abort()
				return
			}

			slog.Error("failed when check JWT token error : %w", err)
			c.JSON(http.StatusUnauthorized, app.Response{
				Code:    app.UnauthorizedErrorCode,
				Message: app.UnauthorizedErrorMessage,
				Data: JWTErrorActionResponse{
					Action: app.ActionLogout,
				},
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*auth.JWTClaims); ok && token.Valid {
			c.Set("userId", claims.UserID)
			c.Next()
		}
	}
}

type JWTErrorActionResponse struct {
	Action string `json:"action"`
}
