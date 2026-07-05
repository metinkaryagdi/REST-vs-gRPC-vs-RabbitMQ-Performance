import os
import json
import matplotlib.pyplot as plt
import pandas as pd

def main():
    json_path = os.path.join("results", "benchmark_results.json")
    if not os.path.exists(json_path):
        print(f"Error: {json_path} not found.")
        return

    with open(json_path, "r", encoding="utf-8") as f:
        data = json.load(f)

    df = pd.DataFrame(data)
    
    os.makedirs("results", exist_ok=True)

    # Style configuration
    plt.style.use('seaborn-v0_8-whitegrid' if 'seaborn-v0_8-whitegrid' in plt.style.available else 'default')
    plt.rcParams['font.family'] = 'sans-serif'
    plt.rcParams['font.sans-serif'] = ['DejaVu Sans', 'Arial', 'Helvetica']
    
    color_map = {
        "rest": "#e74c3c",              # Red
        "grpc": "#2ecc71",              # Green
        "rabbitmq_confirm": "#f39c12",   # Orange
        "rabbitmq_async": "#3498db",     # Blue
        "rabbitmq_e2e": "#9b59b6"        # Purple
    }
    
    label_map = {
        "rest": "REST (JSON / HTTP 1.1)",
        "grpc": "gRPC (Protobuf / HTTP 2)",
        "rabbitmq_confirm": "RabbitMQ (Sync Confirm)",
        "rabbitmq_async": "RabbitMQ (Async Publish)",
        "rabbitmq_e2e": "RabbitMQ End-to-End"
    }

    def plot_metric(payload_size, metric, y_label, title, filename):
        plt.figure(figsize=(10, 6))
        subset = df[df["payload_size"] == payload_size]
        
        modes = subset["mode"].unique()
        for mode in modes:
            mode_data = subset[subset["mode"] == mode].sort_values("concurrency")
            if mode_data.empty:
                continue
            
            plt.plot(
                mode_data["concurrency"], 
                mode_data[metric], 
                marker='o', 
                linewidth=2.5, 
                markersize=8,
                color=color_map.get(mode, "#95a5a6"),
                label=label_map.get(mode, mode)
            )

        plt.title(f"{title} ({payload_size.capitalize()} Payload)", fontsize=14, fontweight='bold', pad=15)
        plt.xlabel("Concurrency (Goroutines)", fontsize=12, labelpad=10)
        plt.ylabel(y_label, fontsize=12, labelpad=10)
        plt.xscale('log')
        
        concurrencies = sorted(subset["concurrency"].unique())
        plt.xticks(concurrencies, labels=[str(c) for c in concurrencies])
        plt.legend(frameon=True, facecolor='white', edgecolor='#e2e8f0', fontsize=10)
        plt.grid(True, linestyle='--', alpha=0.6)
        plt.tight_layout()
        
        output_filepath = os.path.join("results", filename)
        plt.savefig(output_filepath, dpi=300)
        plt.close()
        print(f"Saved: {output_filepath}")

    # Generate plots for small and large payloads
    plot_metric("small", "throughput", "Throughput (RPS)", "Throughput vs Concurrency", "throughput_small.png")
    plot_metric("small", "p95_latency_ms", "95th Percentile Latency (ms)", "P95 Latency vs Concurrency", "latency_p95_small.png")

    plot_metric("large", "throughput", "Throughput (RPS)", "Throughput vs Concurrency", "throughput_large.png")
    plot_metric("large", "p95_latency_ms", "95th Percentile Latency (ms)", "P95 Latency vs Concurrency", "latency_p95_large.png")

if __name__ == "__main__":
    main()
