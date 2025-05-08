package main

import (
	"encoding/json"
	"net/http"
	"time"

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

func createTopicHandler(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateTopicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		streamCfg := &nats.StreamConfig{
			Name:              req.Name,
			Subjects:          []string{req.Subject},
			Storage:           nats.FileStorage,
			Replicas:          1, // 복제본 수 1로 수정됨
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
			http.Error(w, "failed to create stream: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := CreateTopicResponse{
			TopicArn: "srn:scp:sns:kr-cp-1:100000000000:" + req.Name,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func deleteTopicHandler(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topicName := r.URL.Query().Get("name")
		if topicName == "" {
			http.Error(w, "missing 'name' parameter", http.StatusBadRequest)
			return
		}

		err := js.DeleteStream(topicName)
		if err != nil {
			http.Error(w, "failed to delete stream: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Topic deleted successfully"))
	}
}

func listTopicsHandler(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var topics []string
		lister := js.StreamNames()
		for name := range lister {
			arn := "srn:scp:sns:kr-cp-1:100000000000:" + name
			topics = append(topics, arn)
		}

		resp := ListTopicsResponse{Topics: topics}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
