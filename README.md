# API code
```bash
go mod init nats
go get github.com/nats-io/nats.go

# http 서버 실행
go run ./cmd/app/ 
# nats 동기식 publish 성능테스트
go run ./cmd/nats/sync/
# nats 비동기식 publish 성능테스트
go run ./cmd/nats/async/
```

## main.go 
http 단일 진입점

## topic.go
topic 과 관련된 api 

## publish.go
publish 액션과 관련된 api

## subscribe.go
subscribe 액션과 관련된 api 

### 테스트 curl
```bash
# Create API
curl -X POST "http://localhost:8080/v1/?Action=createTopic" \
  -H "Content-Type: application/json" \
  -d '{"name": "sns-wrk-test", "subject": "sns.wrk.test"}'
 
# Delete API
curl -X POST "http://localhost:8080/v1/?Action=deleteTopic&name=sns-wrk-test"

# List API
curl "http://localhost:8080/v1/?Action=listTopics"

# publish
curl -X POST "http://localhost:8080/v1/?Action=publish" \
  -H "Content-Type: application/json" \
  -d '{
        "topicName": "sns-wrk-test",
        "message": "회원가입 이벤트 발생",
        "subject": "sns.wrk.test"
      }'

# publish status check
curl "http://localhost:8080/v1/?Action=publishCheck&messageId=<message-id>"

```

### 부하테스트를 위한 linux 설정 확인
- nats 의 socket connection 테스트 이전에 http 한계를 조절
```bash
# open file descriptors 조절, 리눅스 (파일,소켓, 디바이스, 파이프) 는 파일로 간주
# -c2000 까지 늘렸을 때 Socket erros 가 사라진다. 
ulimit -n 65536

# 기본이 4096으로 되어있음 (local)
sysctl -w net.core.somaxconn=4096

# 기본이 1024 (local)
sysctl -w net.ipv4.tcp_max_syn_backlog=4096

# TIME_WAIT 소켓 재사용 및 FIN 타임아웃 축소
sysctl -w net.ipv4.tcp_fin_timeout=10 # 기본이 60
sysctl -w net.ipv4.tcp_tw_reuse=1 # 기본이 2
```

### 부하테스트
```bash
# -t: 쓰레드,  -c: 커넥션개수, -d: 테스트시간, -H: "Header: value" 로 헤더추가
# --latency : 각 요청의 지연 통계
# -s: <script.lua> 로 custom lua 스크립트 사용. (POST, 헤더설정)
wrk -t10 -c2000 -d10s http://localhost:8080/v1/?Action=listTopics
wrk -t50 -c7000 -d10s -s ~/vscode/nats/lua/publish.lua http://localhost:8080/v1/?Action=publish
```

# NATS-SERVER
## 실행
```nats-server -c nats-server.conf -V```

# nats-client

## 구독
### nats-core 기본 구독
```bash
nats subscribe ">" -s nats://0.0.0.0:4222
```
### consumer 기반 구독 
```bash
# ack 옵션이 있어야 stream 에 발행된 메시지가 push 된 후에 삭제처리가 된다. 
nats subscribe <subject_name> --ack
```

## 발행
### nats-core 기본 발행
```bash
nats pub hello world -s nats://0.0.0.0:4222
```

## 정보생성
연구가 좀 더 필요함. cli 레벨의 옵션

```bash
# stream 생성
nats stream add <stream_name>
# consumer 생성
nats consumer add <stream_name> <consumer_name>
```

## 정보삭제
```bash
# stream 삭제
nats stream rm <stream_name>
```

## 정보조회
```bash
# stream 목록
nats stream ls
# stream 정보
nats stream info <stream_name>
# consumer 목록
nats consumer ls <stream_name>
# consumer 정보
nats consumer info <stream_name> <consumer_name>
```

# VALKEY
```bash
# docker 로 실행
docker run -d --name valkey -p 6379:6379 valkey/valkey
# 이후부터 실행/정지
docker start valkey
docker stop valkey
# cli로 확인
docker exec -it valkey valkey-cli
ping
set foo bar
get foo
```