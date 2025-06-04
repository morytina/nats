package handler

import (
	"nats/internal/service"

	"github.com/labstack/echo/v4"
)

func BuildHandlers(topicSvc service.TopicService, publishSvc service.PublishService) map[string]func() echo.HandlerFunc {
	topicHandler := NewTopicHandler(topicSvc)
	publishHandler := NewPublishHandler(publishSvc)

	return map[string]func() echo.HandlerFunc{
		"createTopic":    topicHandler.Create,
		"deleteTopic":    topicHandler.Delete,
		"listTopics":     topicHandler.List,
		"publish":        publishHandler.Publish,
		"checkAckStatus": publishHandler.CheckAckStatus,
	}
}
