package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	const connNum = 1
	const goroutineNum = 10000
	const messagesPerGoroutine = 10
	const subject = "sns.wrk.test"

	ncPool := make([]*nats.Conn, connNum)
	jsPool := make([]nats.JetStreamContext, connNum)

	for i := 0; i < connNum; i++ {
		nc, _ := nats.Connect(nats.DefaultURL)
		js, _ := nc.JetStream(nats.PublishAsyncMaxPending(65536))
		ncPool[i] = nc
		jsPool[i] = js
	}

	var wg sync.WaitGroup
	var errCnt int64
	start := time.Now()

	for i := 0; i < goroutineNum; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for m := 0; m < messagesPerGoroutine; m++ {
				idx := i % connNum
				js := jsPool[idx]
				_, err := js.PublishAsync(subject, []byte(fmt.Sprintf("msg-%d-%d", id, m)))
				if err != nil {
					atomic.AddInt64(&errCnt, 1)
					fmt.Println("[ERROR]", err)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	totalSent := goroutineNum * messagesPerGoroutine

	for i := 0; i < connNum; i++ {
		js := jsPool[i]
		select {
		case <-js.PublishAsyncComplete():
			fmt.Println("complete")
		case <-time.After(5 * time.Second):
			fmt.Println("Did not resolve in time")
		}
	}

	fmt.Printf("Sent %d messages in %s (%.2f msg/sec), Errors: %d\n",
		totalSent, elapsed, float64(totalSent)/elapsed.Seconds(), errCnt)
}
