package handler

import (
	"net/http"

	"nats/internal/service"
	"nats/pkg/logger"

	"github.com/labstack/echo/v4"
)

type PublishRequest struct {
	TopicName string `json:"topicName"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
}

type PublishResponse struct {
	MessageID string `json:"messageId"`
}

func PublishHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req PublishRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn(ctx, "메시지 요청 파싱 실패", "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		msgID, err := service.PublishMessage(ctx, req.TopicName, req.Message, req.Subject)
		if err != nil {
			logger.Error(ctx, "메시지 발행 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "메시지 발행 성공", "messageId", msgID)
		return c.JSON(http.StatusOK, PublishResponse{MessageID: msgID})
	}
}
