package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func RunRabbitMQBenchmark(config BenchmarkConfig, payload []byte, amqpURL string) MetricResults {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v\n", err)
		return MetricResults{}
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v\n", err)
		return MetricResults{}
	}
	defer ch.Close()

	queueName := "benchmark_queue"
	_, err = ch.QueueDeclare(
		queueName,
		false, // durable
		true,  // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v\n", err)
		return MetricResults{}
	}

	// Purge queue to ensure no dirty state
	_, _ = ch.QueuePurge(queueName, false)

	if config.Mode == "rabbitmq_e2e" {
		return runRabbitMQE2EBenchmark(conn, queueName, config, payload)
	}

	var wg sync.WaitGroup
	latenciesChan := make(chan []time.Duration, config.Concurrency)
	successCountChan := make(chan int, config.Concurrency)

	jobsPerWorker := config.TotalRequests / config.Concurrency
	remainder := config.TotalRequests % config.Concurrency

	startTime := time.Now()

	for w := 0; w < config.Concurrency; w++ {
		wg.Add(1)
		workerRequests := jobsPerWorker
		if w == 0 {
			workerRequests += remainder
		}

		go func(reqCount int) {
			defer wg.Done()

			workerCh, err := conn.Channel()
			if err != nil {
				log.Printf("Worker failed to open channel: %v\n", err)
				return
			}
			defer workerCh.Close()

			if config.Mode == "rabbitmq_confirm" {
				err = workerCh.Confirm(false)
				if err != nil {
					log.Printf("Failed to enable publisher confirms: %v\n", err)
					return
				}
			}

			localLatencies := make([]time.Duration, 0, reqCount)
			localSuccess := 0

			for i := 0; i < reqCount; i++ {
				reqStart := time.Now()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

				if config.Mode == "rabbitmq_confirm" {
					conf, err := workerCh.PublishWithDeferredConfirmWithContext(
						ctx,
						"",        // exchange
						queueName, // key
						false,     // mandatory
						false,     // immediate
						amqp.Publishing{
							ContentType: "application/json",
							Body:        payload,
						},
					)
					if err != nil {
						cancel()
						continue
					}

					// Block until broker confirms message storage/routing
					confirmed := conf.Wait()
					cancel()
					if confirmed {
						localSuccess++
						localLatencies = append(localLatencies, time.Since(reqStart))
					}
				} else {
					// Async - publish fire-and-forget
					err = workerCh.PublishWithContext(
						ctx,
						"",
						queueName,
						false,
						false,
						amqp.Publishing{
							ContentType: "application/json",
							Body:        payload,
						},
					)
					cancel()
					if err == nil {
						localSuccess++
						localLatencies = append(localLatencies, time.Since(reqStart))
					}
				}
			}

			latenciesChan <- localLatencies
			successCountChan <- localSuccess
		}(workerRequests)
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	close(latenciesChan)
	close(successCountChan)

	var allLatencies []time.Duration
	totalSuccess := 0
	for l := range latenciesChan {
		allLatencies = append(allLatencies, l...)
	}
	for s := range successCountChan {
		totalSuccess += s
	}

	// Delete queue to release RabbitMQ resources immediately
	cleanupCh, err := conn.Channel()
	if err == nil {
		_, _ = cleanupCh.QueueDelete(queueName, false, false, false)
		cleanupCh.Close()
	}

	return CalculateMetrics(config, elapsed, allLatencies, totalSuccess)
}

type EventPayloadE2E struct {
	EventID       string `json:"event_id"`
	TimestampNano int64  `json:"timestamp_nano"`
	Payload       string `json:"payload"`
}

func runRabbitMQE2EBenchmark(conn *amqp.Connection, queueName string, config BenchmarkConfig, rawPayload []byte) MetricResults {
	consumeCh, err := conn.Channel()
	if err != nil {
		log.Printf("E2E consumer failed to open channel: %v\n", err)
		return MetricResults{}
	}
	defer consumeCh.Close()

	// Prefetch buffer size tuned for parallel consumption
	_ = consumeCh.Qos(config.Concurrency*10, 0, false)

	deliveries, err := consumeCh.Consume(
		queueName,
		"",    // consumer tag
		true,  // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		log.Printf("E2E consumer failed to consume: %v\n", err)
		return MetricResults{}
	}

	var mu sync.Mutex
	consumerLatencies := make([]time.Duration, 0, config.TotalRequests)
	consumerDone := make(chan struct{})

	go func() {
		for d := range deliveries {
			var msg EventPayloadE2E
			if err := json.Unmarshal(d.Body, &msg); err == nil {
				nowNano := time.Now().UnixNano()
				sentNano := msg.TimestampNano
				if sentNano > 0 {
					latency := time.Duration(nowNano - sentNano)
					mu.Lock()
					consumerLatencies = append(consumerLatencies, latency)
					count := len(consumerLatencies)
					mu.Unlock()

					if count >= config.TotalRequests {
						close(consumerDone)
						break
					}
				}
			}
		}
	}()

	var wg sync.WaitGroup
	jobsPerWorker := config.TotalRequests / config.Concurrency
	remainder := config.TotalRequests % config.Concurrency

	startTime := time.Now()

	for w := 0; w < config.Concurrency; w++ {
		wg.Add(1)
		workerRequests := jobsPerWorker
		if w == 0 {
			workerRequests += remainder
		}

		go func(reqCount int) {
			defer wg.Done()
			workerCh, err := conn.Channel()
			if err != nil {
				return
			}
			defer workerCh.Close()

			for i := 0; i < reqCount; i++ {
				evt := EventPayloadE2E{
					EventID:       fmt.Sprintf("evt_%d", i),
					TimestampNano: time.Now().UnixNano(),
					Payload:       string(rawPayload),
				}
				body, _ := json.Marshal(evt)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = workerCh.PublishWithContext(
					ctx,
					"",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				)
				cancel()
			}
		}(workerRequests)
	}

	wg.Wait()

	select {
	case <-consumerDone:
	case <-time.After(30 * time.Second):
		log.Println("E2E benchmark consumer timed out waiting for messages.")
	}

	elapsed := time.Since(startTime)

	mu.Lock()
	allLatencies := make([]time.Duration, len(consumerLatencies))
	copy(allLatencies, consumerLatencies)
	mu.Unlock()

	// Delete queue to release RabbitMQ resources immediately
	cleanupCh, err := conn.Channel()
	if err == nil {
		_, _ = cleanupCh.QueueDelete(queueName, false, false, false)
		cleanupCh.Close()
	}

	return CalculateMetrics(config, elapsed, allLatencies, len(allLatencies))
}
