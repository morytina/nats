# API code
```bash
go mod init nats
go get github.com/nats-io/nats.go
```
## main.go 
```bash
#실행
go run ./cmd/main.go
```
## topic.go
```bash
#토픽생성
curl -X POST "http://localhost:8080/?Action=createTopic" \
  -H "Content-Type: application/json" \
  -d '{"name": "sns-user-signup", "subject": "sns.user.signup"}'
#토픽삭제
curl -X POST "http://localhost:8080/?Action=deleteTopic&name=sns-user-signup"
#토픽조회
curl "http://localhost:8080/?Action=listTopics"
```

## publish.go
```bash
#테스트
curl -X POST "http://localhost:8080/?Action=publish" \
  -H "Content-Type: application/json" \
  -d '{
        "topicName": "sns-user-signup",
        "message": "회원가입 이벤트 발생",
        "subject": "sns.user.signup"
      }'
```

# nats-server
## 실행
`nats-server -c nats-server.conf -V`

# nats-client
## 체크
`nats server check connection -s nats://0.0.0.0:4222`
## 구독
`nats subscribe ">" -s nats://0.0.0.0:4222`
## 발행
`nats pub hello world -s nats://0.0.0.0:4222`
## 외부서버 구독
`nats sub -s nats://server:port ">"`