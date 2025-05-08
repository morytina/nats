package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nats-io/nats.go"
)

type PublishRequest struct {
	TopicName string `json:"topicName"` // JetStream stream 이름
	Message   string `json:"message"`   // 게시할 본문
	Subject   string `json:"subject"`   // 생략 가능: 기본은 topicName
}

type PublishResponse struct {
	MessageID string `json:"messageId"`
}

func publishHandler(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PublishRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.TopicName == "" || req.Message == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		subject := req.Subject
		if subject == "" {
			subject = req.TopicName
		}

		ack, err := js.Publish(subject, []byte(req.Message))
		if err != nil {
			http.Error(w, "failed to publish: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := PublishResponse{
			MessageID: strconv.FormatUint(ack.Sequence, 10),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
