package main

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type PublishRequest struct {
	TopicName string `json:"topicName"`
	Message   string `json:"message"`
	Subject   string `json:"subject"`
}

type PublishResponse struct {
	MessageID string `json:"messageId"`
}

func publishHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req PublishRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if req.TopicName == "" || req.Message == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		}

		subject := req.Subject
		if subject == "" {
			subject = req.TopicName
		}

		ack, err := js.Publish(subject, []byte(req.Message))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to publish: " + err.Error()})
		}

		resp := PublishResponse{
			MessageID: strconv.FormatUint(ack.Sequence, 10),
		}
		return c.JSON(http.StatusOK, resp)
	}
}
