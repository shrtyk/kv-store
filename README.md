# In-Memory Key-Value Store

[![kv-store-ci](https://github.com/shrtyk/kv-store/actions/workflows/ci.yml/badge.svg)](https://github.com/shrtyk/kv-store/actions/workflows/ci.yml)
[![codecov](https://codecov.io/github/shrtyk/kv-store/graph/badge.svg?token=GVFCB943N5)](https://codecov.io/github/shrtyk/kv-store)
[![Go Report Card](https://goreportcard.com/badge/github.com/shrtyk/kv-store)](https://goreportcard.com/report/github.com/shrtyk/kv-store)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance, persistent, and observable in-memory key-value store built with Go. This project is a deep dive into the internal mechanisms of modern databases, demonstrating a practical exploration of concepts like data persistence, concurrency control, and system observability from the ground up.

## Features

- **Simple HTTP API**: Provides a straightforward RESTful interface for `PUT`, `GET`, and `DELETE` operations.
- **Concurrent & Performant**: Utilizes a sharded map to minimize lock contention, allowing it to handle high-throughput workloads efficiently.
- **Durable Persistence**: Implements a Write-Ahead Log (WAL) using Protocol Buffers to ensure that no data is lost in a compact and efficient binary format.
- **Fast Recovery**: Periodically creates snapshots of the data to compact the WAL, ensuring quick restarts. Snapshots also use Protocol Buffers for efficient storage.
- **Built-in Observability**: Comes with a pre-configured Grafana dashboard for monitoring key performance metrics via Prometheus.

## Architecture and Design Decisions

This project was built with a focus on exploring the core principles behind modern data systems. The following are key architectural decisions made to address common challenges in database design:

### High-Throughput Concurrency

- **Problem**: A single, global lock on a central data map creates a bottleneck under concurrent loads.
- **Solution**: A **sharded map** partitions the key space across many maps, each with its own lock. This distributes write contention, significantly improving parallelism and throughput.

### Crash Safety and Durability

- **Problem**: In-memory data is lost on server crash or restart.
- **Solution**: A **Write-Ahead Log (WAL)** persists all write operations to disk _before_ they are applied in memory. This ensures that the store can be fully recovered by replaying the log after a crash.

### Fast Startup and Recovery

- **Problem**: Replaying a large WAL file on startup can lead to slow recovery times.
- **Solution**: The system periodically creates **snapshots** of the in-memory state. On restart, the service loads the latest snapshot and replays only the WAL entries created since, dramatically reducing startup time.

## Configuration

Before running the application, you need to create a `config.yml` file. A commented example is provided in `config/config.example.yml`. You can copy and modify it to get started:

```bash
cp config/config.example.yml config/config.yml
```

## Getting Started

You can run the application using either Docker Compose or native Go commands.

### With Docker (Recommended)

The easiest way to get started is with Docker Compose, which runs the KV store, Prometheus, and Grafana.

```bash
# Build and start all services in detached mode
docker-compose up -d --build
```

- The KV store will be available at `http://localhost:16700`.
- Prometheus will be available at `http://localhost:9090`.
- Grafana will be available at `http://localhost:3000`.

### With Go

You must have Go installed on your system.

```bash
# Build the binary
make build

# Run the application
# (Requires a config file, one is provided in the `config` directory)
make run

# Run all unit tests
make test

# Run unit tests with coverage
make test-cover

# Run linters
make lint
```

## Observability

The project includes a pre-configured Grafana dashboard for visualizing performance metrics:

- **HTTP Performance**: Request rates, latency distributions (heatmap), and latencypercentiles (P50, P90, P99).
- **Key-Value Operations**: P99 latency for PUT, GET, and DELETE operations.
- **Error Rates**: HTTP 4xx and 5xx error rates.

## Tradeoffs and Future Work

In current implementation, certain tradeoffs were made, prioritizing simplicity and clarity to effectively demonstrate the core concepts.

- **Memory Management**: Go maps do not shrink their memory allocation when items are deleted. For a long-running, write-heavy service, this can lead to high memory usage. A future improvement would be to implement a background process that periodically **rebuilds map shards** to reclaim unused memory.
- **Replication**: To improve availability, a replication mechanism could be added to synchronize data to one or more follower nodes.
- **Clustering**: For horizontal scalability, the store could be extended into a distributed system where data is sharded across multiple nodes in a cluster.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
