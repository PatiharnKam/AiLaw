package sessionshistory

import (
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   SessionService
	validator *validator.Validate
}

func NewHandler(service SessionService) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

func (h *Handler) GetSessionHistory(c *gin.Context) {
	logger := slog.Default()
	var req SessionsHistoryRequest

	req.UserId = c.GetString("userId")

	if err := h.validator.Struct(req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetSessionsHistoryService(ctx, req)
	if err != nil {
		logger.Error("error while get session history : " + err.Error())
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
