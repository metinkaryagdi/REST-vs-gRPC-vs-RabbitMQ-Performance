package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

func RunRESTBenchmark(config BenchmarkConfig, payload []byte, url string) MetricResults {
	// Crucial optimization for benchmarking: configure transport connection pool to prevent socket exhaustion
	tr := &http.Transport{
		MaxIdleConns:        5000,
		MaxIdleConnsPerHost: 5000,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
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
			localLatencies := make([]time.Duration, 0, reqCount)
			localSuccess := 0

			for i := 0; i < reqCount; i++ {
				reqStart := time.Now()
				req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(payload))
				if err != nil {
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				// Always drain and close the body to reuse the TCP connection
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
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
