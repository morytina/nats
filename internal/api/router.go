package api

import (
	"nats/internal/repository"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

// 핸들러 매핑 테이블
var actionHandlers = map[string]func(nats.JetStreamContext) echo.HandlerFunc{
	"createTopic": CreateTopicHandler,
	"deleteTopic": DeleteTopicHandler,
	"listTopics":  ListTopicsHandler,
	"publish":     PublishHandler,
}

func ActionRouter(c echo.Context) error {
	ctx := c.Request().Context()
	js := repository.GetJetStream(ctx)
	action := c.QueryParam("Action")

	if handlerFunc, ok := actionHandlers[action]; ok {
		return handlerFunc(js)(c)
	}
	return c.String(http.StatusBadRequest, "invalid Action")
}
