package handler

import (
	"nats/internal/context/logs"
	"nats/internal/entity"
	"nats/internal/service"
	"net/http"
	"strings"

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
	Name string `json:"Name" validate:"required"`
}

type CreateTopicResponse struct {
	CreateTopicResult entity.Topic            `json:"CreateTopicResult"`
	ResponseMetadata  entity.ResponseMetadata `json:"ResponseMetadata"`
}

type DeleteTopicRequest struct {
	TopicSrn string `json:"TopicSrn" validate:"required"`
}

type DeleteTopicResponse struct {
	ResponseMetadata entity.ResponseMetadata `json:"ResponseMetadata"`
}

type ListTopicsResponse struct {
	Topics []entity.Topic `json:"topics"`
}

func (h *TopicHandler) Create() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req CreateTopicRequest
		if err := c.Bind(&req); err != nil {
			logs.GetLogger(ctx).Error("Invalid createTopic request parameter", zap.Error(err))
			return c.JSON(entity.InvalidParameter.HTTPCode, entity.InvalidParameter.Error)
		}

		if err := c.Validate(&req); err != nil {
			logs.GetLogger(ctx).Error("Required parameter is missing", zap.Error(err))
			return c.JSON(entity.InvalidParameter.HTTPCode, entity.InvalidParameter.Error)
		}

		result, err := h.svc.CreateTopic(ctx, req.Name, c.Param("accountid"))
		if err != nil {
			logs.GetLogger(ctx).Error("Failed to create stream", zap.Error(err))
			return c.JSON(entity.InternalError.HTTPCode, entity.InternalError.Error)
		}
		logs.GetLogger(ctx).Info("Stream creation success", zap.String("topic", req.Name))
		meta := entity.ResponseMetadata{RequestId: c.Response().Header().Get(echo.HeaderXRequestID)}
		return c.JSON(http.StatusOK, CreateTopicResponse{
			CreateTopicResult: result, ResponseMetadata: meta,
		})
	}
}

func (h *TopicHandler) Delete() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		var req DeleteTopicRequest
		if err := c.Bind(&req); err != nil {
			logs.GetLogger(ctx).Error("Invalid deleteTopic request parameter", zap.Error(err))
			return c.JSON(entity.InvalidParameter.HTTPCode, entity.InvalidParameter.Error)
		}

		if err := c.Validate(&req); err != nil {
			logs.GetLogger(ctx).Error("Required parameter is missing", zap.Error(err))
			return c.JSON(entity.InvalidParameter.HTTPCode, entity.InvalidParameter.Error)
		}

		parts := strings.Split(req.TopicSrn, ":")
		name := parts[len(parts)-1]

		if name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'name' parameter"})
		}

		if err := h.svc.DeleteTopic(ctx, name); err != nil {
			logs.GetLogger(ctx).Error("Failed to delete stream", zap.Error(err))
			return c.JSON(entity.InternalError.HTTPCode, entity.InternalError.Error)
		}

		logs.GetLogger(ctx).Info("Stream deletion success", zap.String("topic", name))
		meta := entity.ResponseMetadata{RequestId: c.Response().Header().Get(echo.HeaderXRequestID)}
		return c.JSON(http.StatusOK, DeleteTopicResponse{ResponseMetadata: meta})
	}
}

func (h *TopicHandler) List() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		topics, err := h.svc.ListTopics(ctx, c.Param("accountid"))
		if err != nil {
			logs.GetLogger(ctx).Error("Topic list lookup failed", zap.Error(err))
			return c.JSON(entity.InternalError.HTTPCode, entity.InternalError.Error)
		}

		logs.GetLogger(ctx).Info("Return topic list", zap.Int("count", len(topics)))
		return c.JSON(http.StatusOK, ListTopicsResponse{Topics: topics})
	}
}
