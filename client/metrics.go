package main

import (
	"sort"
	"time"
)

type BenchmarkConfig struct {
	Mode          string `json:"mode"`
	Concurrency   int    `json:"concurrency"`
	TotalRequests int    `json:"total_requests"`
	PayloadSize   string `json:"payload_size"`
}

type MetricResults struct {
	Mode          string  `json:"mode"`
	Concurrency   int     `json:"concurrency"`
	PayloadSize   string  `json:"payload_size"`
	TotalRequests int     `json:"total_requests"`
	TotalTimeSec  float64 `json:"total_time_sec"`
	Throughput    float64 `json:"throughput"`
	MeanLatencyMs float64 `json:"mean_latency_ms"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P90LatencyMs  float64 `json:"p90_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	SuccessRate   float64 `json:"success_rate"`
}

func CalculateMetrics(config BenchmarkConfig, elapsed time.Duration, latencies []time.Duration, successCount int) MetricResults {
	total := len(latencies)
	if total == 0 {
		return MetricResults{
			Mode:          config.Mode,
			Concurrency:   config.Concurrency,
			PayloadSize:   config.PayloadSize,
			TotalRequests: config.TotalRequests,
		}
	}

	// Sort latencies to compute percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum int64
	for _, lat := range latencies {
		sum += lat.Nanoseconds()
	}

	mean := float64(sum) / float64(total) / 1e6 // Convert nanoseconds to milliseconds

	p50Idx := int(float64(total) * 0.50)
	if p50Idx >= total {
		p50Idx = total - 1
	}
	p50 := float64(latencies[p50Idx].Nanoseconds()) / 1e6

	p90Idx := int(float64(total) * 0.90)
	if p90Idx >= total {
		p90Idx = total - 1
	}
	p90 := float64(latencies[p90Idx].Nanoseconds()) / 1e6

	p95Idx := int(float64(total) * 0.95)
	if p95Idx >= total {
		p95Idx = total - 1
	}
	p95 := float64(latencies[p95Idx].Nanoseconds()) / 1e6

	p99Idx := int(float64(total) * 0.99)
	if p99Idx >= total {
		p99Idx = total - 1
	}
	p99 := float64(latencies[p99Idx].Nanoseconds()) / 1e6

	throughput := float64(successCount) / elapsed.Seconds()
	successRate := (float64(successCount) / float64(config.TotalRequests)) * 100.0

	return MetricResults{
		Mode:          config.Mode,
		Concurrency:   config.Concurrency,
		PayloadSize:   config.PayloadSize,
		TotalRequests: config.TotalRequests,
		TotalTimeSec:  elapsed.Seconds(),
		Throughput:    throughput,
		MeanLatencyMs: mean,
		P50LatencyMs:  p50,
		P90LatencyMs:  p90,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		SuccessRate:   successRate,
	}
}
