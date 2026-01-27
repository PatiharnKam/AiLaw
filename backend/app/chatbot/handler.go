package chatbot

import (
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   Service
	validator *validator.Validate
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

func (h *Handler) CreateChatSessionHandler(c *gin.Context) {
	logger := slog.Default()
	var req CreateChatSessionRequest

	req.UserId = c.GetString("userId")

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.CreateChatSessionService(ctx, req)
	if err != nil {
		logger.Error("error while get message history : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data:    resp.SessionId,
	})
}

func (h *Handler) ChatbotProcessModelHandler(c *gin.Context) {
	logger := slog.Default()
	var req ChatbotProcessRequest

	req.UserId = c.GetString("userId")

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	if err := h.validator.Struct(req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.ChatbotProcess(ctx, req)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data:    resp,
	})
}
