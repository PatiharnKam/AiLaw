package deletesession

import (
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   DeleteChatSessionService
	validator *validator.Validate
}

func NewHandler(service DeleteChatSessionService) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

func (h *Handler) DeleteChatSessionHandler(c *gin.Context) {
	logger := slog.Default()
	var req DeleteChatSessionRequest

	req.UserID = c.GetString("userId")
	req.SessionID = c.Param("sessionID")

	if err := h.validator.Struct(req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	ctx := c.Request.Context()
	err := h.service.DeleteChatSessionService(ctx, req)
	if err != nil {
		logger.Error("error while delete session history : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
	})
}
