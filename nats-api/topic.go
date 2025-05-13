package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type CreateTopicRequest struct {
	Name    string `json:"name"`
	Subject string `json:"subject"`
}

type CreateTopicResponse struct {
	TopicArn string `json:"topicArn"`
}

type ListTopicsResponse struct {
	Topics []string `json:"topics"`
}

func createTopicHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req CreateTopicRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		streamCfg := &nats.StreamConfig{
			Name:              req.Name,
			Subjects:          []string{req.Subject},
			Storage:           nats.FileStorage,
			Replicas:          1,
			Retention:         nats.LimitsPolicy,
			Discard:           nats.DiscardOld,
			MaxMsgs:           -1,
			MaxMsgsPerSubject: -1,
			MaxBytes:          -1,
			MaxAge:            96 * time.Hour,
			MaxMsgSize:        262144,
			Duplicates:        0,
			AllowRollup:       false,
			DenyDelete:        false,
			DenyPurge:         false,
		}

		_, err := js.AddStream(streamCfg)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create stream: " + err.Error(),
			})
		}

		resp := CreateTopicResponse{
			TopicArn: "srn:scp:sns:kr-cp-1:100000000000:" + req.Name,
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func deleteTopicHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		topicName := c.QueryParam("name")
		if topicName == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing 'name' parameter"})
		}

		err := js.DeleteStream(topicName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to delete stream: " + err.Error(),
			})
		}

		return c.String(http.StatusOK, "Topic deleted successfully")
	}
}

func listTopicsHandler(js nats.JetStreamContext) echo.HandlerFunc {
	return func(c echo.Context) error {
		var topics []string
		lister := js.StreamNames()

		for name := range lister {
			arn := "srn:scp:sns:kr-cp-1:100000000000:" + name
			topics = append(topics, arn)
		}

		resp := ListTopicsResponse{Topics: topics}
		return c.JSON(http.StatusOK, resp)
	}
}
