package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// 핸들러 매핑 테이블
var actionHandlers = map[string]func() echo.HandlerFunc{
	"createTopic": CreateTopicHandler,
	"deleteTopic": DeleteTopicHandler,
	"listTopics":  ListTopicsHandler,
	"publish":     PublishHandler,
}

func ActionRouter(c echo.Context) error {
	action := c.QueryParam("Action")
	if handlerFunc, ok := actionHandlers[action]; ok {
		return handlerFunc()(c)
	}
	return c.String(http.StatusBadRequest, "invalid Action")
}
