package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"nats/internal/context/metrics"
)

// 핸들러 매핑 테이블
var actionHandlers = map[string]func() echo.HandlerFunc{
	"createTopic":  CreateTopicHandler,
	"deleteTopic":  DeleteTopicHandler,
	"listTopics":   ListTopicsHandler,
	"publish":      PublishHandler,
	"publishCheck": CheckAckStatusHandler,
}

func ActionRouter(c echo.Context) error {
	action := c.QueryParam("Action")
	if handlerFunc, ok := actionHandlers[action]; ok {
		metrics.ApiCallCounter.WithLabelValues(action).Inc()
		return handlerFunc()(c)
	}
	return c.String(http.StatusBadRequest, "invalid Action")
}
