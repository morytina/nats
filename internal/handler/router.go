package handler

import (
	"net/http"
	"strconv"

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
		err := handlerFunc()(c)
		metrics.ApiCallCounter.WithLabelValues(action, strconv.Itoa(c.Response().Status)).Inc()
		return err
	}
	return c.String(http.StatusBadRequest, "invalid Action")
}
