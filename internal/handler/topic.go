package handler

import (
	"nats/internal/context/logs"
	"nats/internal/entity"
	"nats/internal/service"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type TopicHandler struct {
	svc service.TopicService
}

func NewTopicHandler(svc service.TopicService) *TopicHandler {
	return &TopicHandler{svc: svc}
}

type CreateTopicRequest struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

type CreateTopicResponse struct {
	TopicArn string `json:"topicArn"`
}

type ListTopicsResponse struct {
	Topics []entity.Topic `json:"topics"`
}

func (h *TopicHandler) Create() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger := logs.GetLogger(ctx)

		var req CreateTopicRequest
		if err := c.Bind(&req); err != nil {
			logger.Warn("요청 파싱 실패", zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if err := h.svc.CreateTopic(ctx, req.Name, req.Subject); err != nil {
			logger.Error("스트림 생성 실패", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info("스트림 생성 성공", zap.String("topic", req.Name))
		return c.JSON(http.StatusOK, CreateTopicResponse{
			TopicArn: "srn:scp:sns:kr-cp-1:100000000000:" + req.Name,
		})
	}
}

func (h *TopicHandler) Delete() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger := logs.GetLogger(ctx)

		name := c.QueryParam("name")
		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'name' parameter"})
		}

		if err := h.svc.DeleteTopic(ctx, name); err != nil {
			logger.Error("스트림 삭제 실패", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info("스트림 삭제 성공", zap.String("topic", name))
		return c.String(http.StatusOK, name+" topic deleted successfully\n")
	}
}

func (h *TopicHandler) List() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		logger := logs.GetLogger(ctx)

		accountID := c.Param("accountid")
		topics, err := h.svc.ListTopics(ctx, accountID)
		if err != nil {
			logger.Error("리스트 조회 실패", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		logger.Info("리스트 반환", zap.Int("count", len(topics)))
		return c.JSON(http.StatusOK, ListTopicsResponse{Topics: topics})
	}
}
