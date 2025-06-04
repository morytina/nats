package handler

import (
	"nats/internal/context/logs"
	"nats/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type PublishHandler struct {
	svc service.PublishService
}

func NewPublishHandler(svc service.PublishService) *PublishHandler {
	return &PublishHandler{svc: svc}
}

type PublishRequest struct {
	TopicName string `json:"topicName"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
}

type PublishResponse struct {
	MessageID string `json:"messageId"`
}

func (h *PublishHandler) Publish() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger := logs.GetLogger(ctx)

		var req PublishRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn("메시지 요청 파싱 실패", zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		msgID, err := h.svc.PublishAsyncMessage(ctx, req.TopicName, req.Message, req.Subject)
		if err != nil {
			logger.Error("메시지 발행 실패", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info("메시지 발행 성공", zap.String("messageId", msgID))
		return c.JSON(http.StatusOK, PublishResponse{MessageID: msgID})
	}
}

func (h *PublishHandler) CheckAckStatus() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger := logs.GetLogger(ctx)

		id := c.QueryParam("messageId")
		if id == "" {
			logger.Warn("ack 조회 요청에 ID 없음")
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing message id"})
		}

		status, err := h.svc.CheckAckStatus(ctx, id)
		if err != nil {
			logger.Warn("ack 상태 조회 실패", zap.String("id", id), zap.Error(err))
			return c.JSON(http.StatusNotFound, map[string]string{"error": "message id not found"})
		}

		logger.Info("ack 상태 조회 성공", zap.String("id", id), zap.String("status", status))
		return c.JSON(http.StatusOK, map[string]string{"status": status})
	}
}
