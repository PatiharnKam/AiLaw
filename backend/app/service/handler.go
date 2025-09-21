package service

import (
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetMessage(c *gin.Context) {
	logger := slog.Default()
	var req GetMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    "88888",
			Message: "invalid request body : " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetMessageService(ctx, req)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    "99999",
			Message: "error from service layer" + err.Error(),
		})
		return
	}

	logger.Info("success")
	c.JSON(http.StatusOK, app.Response{
		Code:    "00000",
		Message: "SUCCESS",
		Data:    resp,
	})
}

func (h *Handler) GetMessageModel(c *gin.Context) {
	logger := slog.Default()
	var req GetMessageModelRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("invalid request body : " + err.Error())
		c.JSON(http.StatusBadRequest, app.Response{
			Code:    "88888",
			Message: "invalid request body : " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.service.GetMessageModelService(ctx, req)
	if err != nil {
		logger.Error("error from service layer : " + err.Error())
		c.JSON(http.StatusInternalServerError, app.Response{
			Code:    "99999",
			Message: "error from service layer" + err.Error(),
		})
		return
	}

	logger.Info("success")
	c.JSON(http.StatusOK, app.Response{
		Code:    "00000",
		Message: "SUCCESS",
		Data:    resp,
	})
}
