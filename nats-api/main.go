package main

import (
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
)

const (
	cspRegion  = "kr-cp-1"
	cspAccount = "100000000000"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("NATS 연결 실패: %v", err)
	}
	defer nc.Drain()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("JetStream 컨텍스트 생성 실패: %v", err)
	}

	http.HandleFunc("/", ActionRouter(js))
	log.Println("API 서버 실행 중: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ActionRouter(js nats.JetStreamContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("Action")
		switch action {
		case "createTopic":
			createTopicHandler(js)(w, r)
		case "deleteTopic":
			deleteTopicHandler(js)(w, r)
		case "listTopics":
			listTopicsHandler(js)(w, r)
		case "publish":
			publishHandler(js)(w, r)
		default:
			http.Error(w, "invalid Action", http.StatusBadRequest)
		}
	}
}
