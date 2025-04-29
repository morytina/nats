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