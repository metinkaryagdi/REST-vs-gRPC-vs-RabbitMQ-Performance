package main

import (
	"context"
	"sync"
	"time"

	pb "benchmark/server/eventpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RunGRPCBenchmark(config BenchmarkConfig, req *pb.LogRequest, addr string) MetricResults {
	// Setup gRPC connection options for high performance
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return MetricResults{}
	}
	defer conn.Close()

	client := pb.NewEventServiceClient(conn)

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
			localLatencies := make([]time.Duration, 0, reqCount)
			localSuccess := 0

			for i := 0; i < reqCount; i++ {
				reqStart := time.Now()
				// Run blocking RPC call concurrently over the shared HTTP/2 connection
				_, err := client.LogEvent(context.Background(), req)
				if err == nil {
					localSuccess++
					localLatencies = append(localLatencies, time.Since(reqStart))
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

	return CalculateMetrics(config, elapsed, allLatencies, totalSuccess)
}
