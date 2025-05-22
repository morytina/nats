package handler

import (
	"net/http"

	"nats/internal/service"
	"nats/pkg/logger"

	"github.com/labstack/echo/v4"
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

func CreateTopicHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req CreateTopicRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn(ctx, "요청 파싱 실패", "error", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		err := service.CreateTopic(ctx, req.Name, req.Subject)
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

func DeleteTopicHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		name := c.QueryParam("name")
		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'name' parameter"})
		}

		err := service.DeleteTopic(ctx, name)
		if err != nil {
			logger.Error(ctx, "스트림 삭제 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "스트림 삭제 성공", "topic", name)
		return c.String(http.StatusOK, "Topic deleted successfully")
	}
}

func ListTopicsHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		topics, err := service.ListTopics(ctx)
		if err != nil {
			logger.Error(ctx, "리스트 조회 실패", "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info(ctx, "리스트 반환", "count", len(topics))
		return c.JSON(http.StatusOK, ListTopicsResponse{Topics: topics})
	}
}
