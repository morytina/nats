package handler

import (
	"nats/internal/service"

	"github.com/labstack/echo/v4"
)

func AccountBaseHandlers(topicSvc service.TopicService) map[string]func() echo.HandlerFunc {
	topicHandler := NewTopicHandler(topicSvc)

	return map[string]func() echo.HandlerFunc{
		"createTopic": topicHandler.Create,
		"listTopics":  topicHandler.List,
	}
}

func AccountTopicBaseHandlers(topicSvc service.TopicService, publishSvc service.PublishService) map[string]func() echo.HandlerFunc {
	topicHandler := NewTopicHandler(topicSvc)
	publishHandler := NewPublishHandler(publishSvc)

	return map[string]func() echo.HandlerFunc{
		"deleteTopic":  topicHandler.Delete,
		"publish":      publishHandler.Publish,
		"publishCheck": publishHandler.CheckAckStatus,
	}
}
