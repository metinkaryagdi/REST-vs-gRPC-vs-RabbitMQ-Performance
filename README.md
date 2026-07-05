# REST vs gRPC vs RabbitMQ Performance Comparison

A high-performance benchmark suite built in **Go** to measure and analyze throughput, latency, and payload size characteristics across three common service communication protocols: **REST (JSON over HTTP/1.1)**, **gRPC (Protobuf over HTTP/2)**, and **RabbitMQ (AMQP 0-9-1)**.

For a detailed deep-dive analysis, check the comprehensive technical article in Turkish: [blog_post.md](blog_post.md).

## Key Findings

- **gRPC is the Throughput King:** Achieved over **100,000 RPS** (Requests Per Second) under concurrent load with small payloads, outperforming REST by over **12x** due to HTTP/2 connection multiplexing and Protocol Buffers serialization.
- **RabbitMQ Async is Ultra-Fast:** Reached up to **52,000 Msg/sec** publishing rates in fire-and-forget mode.
- **REST is CPU & Connection Bound:** Maxed out at **~7,900 RPS** due to HTTP/1.1 head-of-line blocking and JSON serialization CPU overhead.

## Architecture

- **`server/`:** Implements REST (Go standard library `net/http`) and gRPC servers in parallel with graceful shutdown.
- **`client/`:** Implements multi-concurrency benchmark clients for HTTP REST, gRPC, and RabbitMQ (supporting sync publisher confirms, async publishes, and End-to-End consumption latency tracking).
- **`scripts/`:** PowerShell automation suite and Python plotting utility.

## Performance Visualizations

### Small Payload (~250 bytes)
![Throughput Small](results/throughput_small.png)
![Latency Small](results/latency_p95_small.png)

### Large Payload (~10 KB)
![Throughput Large](results/throughput_large.png)
![Latency Large](results/latency_p95_large.png)

## Getting Started

### Prerequisites
- Go 1.26+
- Docker & Docker Compose
- Python 3 (with `pandas` and `matplotlib` installed for plotting)

### Running Benchmarks
Simply run the automated PowerShell script. It will spin up RabbitMQ via Docker, compile Go binaries, run all benchmark matrices, teardown resources, and generate charts:
```powershell
powershell -ExecutionPolicy Bypass -File scripts/run_benchmarks.ps1
```

## License
MIT License
