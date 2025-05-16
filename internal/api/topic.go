package api

import (
	"net/http"
	"time"

	"nats/internal/logger"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type CreateTopicRequest struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

type CreateTopicResponse struct {
	TopicArn string `json:"topicArn"`
}

type ListTopicsResponse struct {
	Topics []string `json:"topics"`
}

func CreateTopicHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req CreateTopicRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn(ctx, "요청 파싱 실패", "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		streamCfg := &nats.StreamConfig{
			Name:              req.Name,
			Subjects:          []string{req.Subject},
			Storage:           nats.FileStorage,
			Replicas:          1,
			Retention:         nats.LimitsPolicy,
			Discard:           nats.DiscardOld,
			MaxMsgs:           -1,
			MaxMsgsPerSubject: -1,
			MaxBytes:          -1,
			MaxAge:            96 * time.Hour,
			MaxMsgSize:        262144,
			Duplicates:        0,
			AllowRollup:       false,
			DenyDelete:        false,
			DenyPurge:         false,
		}

		_, err := js.AddStream(streamCfg)
		if err != nil {
			logger.Error(ctx, "스트림 생성 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "스트림 생성 성공", "topic", req.Name)
		return c.JSON(http.StatusOK, CreateTopicResponse{
			TopicArn: "srn:scp:sns:kr-cp-1:100000000000:" + req.Name,
		})
	}
}

func DeleteTopicHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		name := c.QueryParam("name")
		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'name' parameter"})
		}

		if err := js.DeleteStream(name); err != nil {
			logger.Error(ctx, "스트림 삭제 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "스트림 삭제 성공", "topic", name)
		return c.String(http.StatusOK, "Topic deleted successfully")
	}
}

func ListTopicsHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var topics []string
		lister := js.StreamNames()
		for name := range lister {
			arn := "srn:scp:sns:kr-cp-1:100000000000:" + name
			topics = append(topics, arn)
		}

		logger.Info(ctx, "리스트 반환", "count", len(topics))
		return c.JSON(http.StatusOK, ListTopicsResponse{Topics: topics})
	}
}
