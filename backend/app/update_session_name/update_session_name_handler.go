package updatesessionname

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service   UpdateSessionNameService
	validator *validator.Validate
}

func NewHandler(service UpdateSessionNameService) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

func (h *Handler) UpdateSessionNameHandler(c *gin.Context) {
	logger := slog.Default()
	var req UpdateSessionNameRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		fmt.Println(req)
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    app.InvalidRequestErrorCode,
			Message: app.InvalidRequestErrorMessage,
		})
		return
	}

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
	err := h.service.UpdateSessionNameService(ctx, req)
	if err != nil {
		logger.Error("error while update session name : " + err.Error())
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
