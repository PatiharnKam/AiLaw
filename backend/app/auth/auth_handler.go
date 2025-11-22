package auth

import (
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service AuthService
}

func NewHandler(service AuthService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetToken(c *gin.Context) {
	ctx := c.Request.Context()
	resp, err := h.service.GetToken(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data:    resp,
	})
}

func (h *Handler) GoogleLogin(c *gin.Context) {
	email := c.Query("email")
	loginURL := h.service.GetGoogleLoginURL(email)

	c.Redirect(http.StatusTemporaryRedirect, loginURL)
}

func (h *Handler) GoogleCallback(c *gin.Context) {
	logger := slog.Default()
	var req GoogleCallbackRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	resp, err := h.service.HandleGoogleCallback(c.Request.Context(), req)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		30*24*60*60, // 30 days
		"/auth",     // จำกัดเฉพาะ refresh endpoint
		"",
		true, // secure=true ใน production
		true, // httpOnly=true
	)

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data: map[string]interface{}{
			"accessToken": resp.AccessToken,
			"userId":      resp.UserId,
		},
	})
}

func (h *Handler) RefreshTokenProcess(c *gin.Context) {
	logger := slog.Default()

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		logger.Error("refresh token not found : " + err.Error())
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	resp, err := h.service.RefreshTokenService(c.Request.Context(), refreshToken)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		resp.RefreshToken,
		30*24*60*60, // 30 days
		// "/auth/refresh", // จำกัดเฉพาะ refresh endpoint
		"/auth", // จำกัดเฉพาะ refresh endpoint
		"",
		true, // secure=true ใน production
		true, // httpOnly=true
	)

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data: map[string]interface{}{
			"accessToken": resp.AccessToken,
			"user":        resp.UserId,
		},
	})

}

func (h *Handler) Logout(c *gin.Context) {
	logger := slog.Default()

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		logger.Error("refresh token not found : " + err.Error())
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}
	err = h.service.LogoutProcess(c.Request.Context(), refreshToken)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}

	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/auth",
		"",
		true,
		true,
	)

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
	})
}
