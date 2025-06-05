package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"nats/internal/context/logs"
	"nats/internal/context/metrics"
)

type ApiRouter interface {
	Register(g *echo.Group)
}

type apiRouter struct {
	accountBaseHandlers      map[string]func() echo.HandlerFunc
	accountTopicBaseHandlers map[string]func() echo.HandlerFunc
}

func NewApiRouter(accountBaseHandlers map[string]func() echo.HandlerFunc, accountTopicBaseHandlers map[string]func() echo.HandlerFunc) ApiRouter {
	return &apiRouter{accountBaseHandlers: accountBaseHandlers, accountTopicBaseHandlers: accountTopicBaseHandlers}
}

func (r *apiRouter) Register(g *echo.Group) {
	g.Any("/:accountid", r.handleAccountBase)
	g.Any("/:accountid/:topicid", r.handleAccountTopicBase)
}

func (r *apiRouter) handleAccountBase(c echo.Context) error {
	logs.GetLogger(c.Request().Context()).Info("handleAccountBase")
	action := c.QueryParam("Action")

	if handlerFunc, ok := r.accountBaseHandlers[action]; ok {
		err := handlerFunc()(c)
		metrics.ApiCallCounter.WithLabelValues(action, strconv.Itoa(c.Response().Status)).Inc()
		return err
	}

	metrics.ApiCallCounter.WithLabelValues(action, "400").Inc()
	return c.String(http.StatusBadRequest, "invalid Action")
}

func (r *apiRouter) handleAccountTopicBase(c echo.Context) error {
	logs.GetLogger(c.Request().Context()).Info("handleAccountTopicBase")
	action := c.QueryParam("Action")

	if handlerFunc, ok := r.accountTopicBaseHandlers[action]; ok {
		err := handlerFunc()(c)
		metrics.ApiCallCounter.WithLabelValues(action, strconv.Itoa(c.Response().Status)).Inc()
		return err
	}

	metrics.ApiCallCounter.WithLabelValues(action, "400").Inc()
	return c.String(http.StatusBadRequest, "invalid Action")
}
