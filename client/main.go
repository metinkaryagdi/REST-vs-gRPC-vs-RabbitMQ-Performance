package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "benchmark/server/eventpb"
)

func makeLargeString(size int) string {
	var builder strings.Builder
	builder.Grow(size)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < size; i++ {
		builder.WriteByte(chars[i%len(chars)])
	}
	return builder.String()
}

func main() {
	mode := flag.String("mode", "rest", "Benchmark mode (rest, grpc, rabbitmq_confirm, rabbitmq_async, rabbitmq_e2e)")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	requests := flag.Int("requests", 10000, "Total number of requests to execute")
	payloadSize := flag.String("payload-size", "small", "Payload size (small, large)")
	addr := flag.String("addr", "", "Address of the service (default depends on mode)")
	outputFile := flag.String("output", "", "JSON file to append results")
	flag.Parse()

	serviceAddr := *addr
	if serviceAddr == "" {
		switch *mode {
		case "rest":
			serviceAddr = "http://127.0.0.1:8080/log"
		case "grpc":
			serviceAddr = "127.0.0.1:50051"
		case "rabbitmq_confirm", "rabbitmq_async", "rabbitmq_e2e":
			serviceAddr = "amqp://guest:guest@127.0.0.1:5672/"
		}
	}

	config := BenchmarkConfig{
		Mode:          *mode,
		Concurrency:   *concurrency,
		TotalRequests: *requests,
		PayloadSize:   *payloadSize,
	}

	fmt.Printf("\n--- Running Benchmark ---\nMode: %s | Concurrency: %d | Requests: %d | Payload: %s | Target: %s\n",
		config.Mode, config.Concurrency, config.TotalRequests, config.PayloadSize, serviceAddr)

	var results MetricResults

	var merchantVal string
	if config.PayloadSize == "large" {
		merchantVal = makeLargeString(10 * 1024) // 10KB string
	} else {
		merchantVal = "amazon"
	}

	// Prepare payload for REST & RabbitMQ JSON comparison
	restPayloadMap := map[string]interface{}{
		"event_id":   "893c5c9a-b249-4171-8bc6-94672e81ea78",
		"timestamp":  time.Now().Unix(),
		"user_id":    "usr_98a72c",
		"event_type": "transaction",
		"payload": map[string]interface{}{
			"amount":   120.50,
			"currency": "USD",
			"merchant": merchantVal,
			"location": "US-NY",
			"device":   "mobile",
		},
	}
	rawPayload, _ := json.Marshal(restPayloadMap)

	switch config.Mode {
	case "rest":
		results = RunRESTBenchmark(config, rawPayload, serviceAddr)
	case "grpc":
		grpcReq := &pb.LogRequest{
			EventId:   "893c5c9a-b249-4171-8bc6-94672e81ea78",
			Timestamp: time.Now().Unix(),
			UserId:    "usr_98a72c",
			EventType: "transaction",
			Payload: &pb.EventPayload{
				Amount:   120.50,
				Currency: "USD",
				Merchant: merchantVal,
				Location: "US-NY",
				Device:   "mobile",
			},
		}
		results = RunGRPCBenchmark(config, grpcReq, serviceAddr)
	case "rabbitmq_confirm", "rabbitmq_async", "rabbitmq_e2e":
		results = RunRabbitMQBenchmark(config, rawPayload, serviceAddr)
	default:
		log.Fatalf("Unknown mode: %s\n", config.Mode)
	}

	fmt.Printf("Results:\n")
	fmt.Printf("  Time Taken:    %.3f seconds\n", results.TotalTimeSec)
	fmt.Printf("  Throughput:    %.1f req/sec\n", results.Throughput)
	fmt.Printf("  Avg Latency:   %.3f ms\n", results.MeanLatencyMs)
	fmt.Printf("  P50 Latency:   %.3f ms\n", results.P50LatencyMs)
	fmt.Printf("  P95 Latency:   %.3f ms\n", results.P95LatencyMs)
	fmt.Printf("  P99 Latency:   %.3f ms\n", results.P99LatencyMs)
	fmt.Printf("  Success Rate:  %.1f%%\n", results.SuccessRate)

	if *outputFile != "" {
		var list []MetricResults
		if _, err := os.Stat(*outputFile); err == nil {
			f, err := os.Open(*outputFile)
			if err == nil {
				dec := json.NewDecoder(f)
				_ = dec.Decode(&list)
				f.Close()
			}
		}

		list = append(list, results)

		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v\n", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(list); err != nil {
			log.Fatalf("Failed to write JSON: %v\n", err)
		}
		fmt.Printf("Saved results to %s\n", *outputFile)
	}
}
