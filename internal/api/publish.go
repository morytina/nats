package api

import (
	"net/http"
	"strconv"

	"nats/internal/logger"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type PublishRequest struct {
	TopicName string `json:"topicName"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
}

type PublishResponse struct {
	MessageID string `json:"messageId"`
}

func PublishHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req PublishRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn("메시지 요청 파싱 실패", "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if req.TopicName == "" || req.Message == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		}

		subject := req.Subject
		if subject == "" {
			subject = req.TopicName
		}

		ack, err := js.Publish(subject, []byte(req.Message))
		if err != nil {
			logger.Error("메시지 발행 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info("메시지 발행 성공", "subject", subject, "sequence", ack.Sequence)
		return c.JSON(http.StatusOK, PublishResponse{
			MessageID: strconv.FormatUint(ack.Sequence, 10),
		})
	}
}
