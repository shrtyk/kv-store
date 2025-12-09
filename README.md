# In-Memory Key-Value Store

[![kv-store-ci](https://github.com/shrtyk/kv-store/actions/workflows/ci.yml/badge.svg)](https://github.com/shrtyk/kv-store/actions/workflows/ci.yml)
[![codecov](https://codecov.io/github/shrtyk/kv-store/graph/badge.svg?token=GVFCB943N5)](https://codecov.io/github.com/shrtyk/kv-store)
[![Go Report Card](https://goreportcard.com/badge/github.com/shrtyk/kv-store)](https://goreportcard.com/report/github.com/shrtyk/kv-store)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance, distributed, persistent, and observable in-memory key-value store built with Go. This project started as a single-node store and has been migrated to a distributed system using a custom-built [Raft consensus library](https://github.com/shrtyk/raft-core). It's a deep dive into the internal mechanisms of modern distributed databases.

## Features

- **Distributed Consensus**: Achieves high availability and strong consistency using the Raft consensus algorithm.
- **Automatic Leader Election & Data Replication**: Tolerates node failures and maintains data consistency across the cluster.
- **Dual HTTP & gRPC APIs**: Interact via a simple RESTful interface or a high-performance gRPC API for `PUT`, `GET`, and `DELETE` operations. Client requests are automatically redirected to the cluster leader.
- **Concurrent & Performant**: Utilizes a sharded map to minimize lock contention, allowing it to handle high-throughput workloads efficiently.
- **Durable Persistence**: The Raft log ensures that all committed operations are durable and can be recovered after a crash.
- **Built-in Observability**: Comes with a pre-configured Grafana dashboard for monitoring key performance metrics via Prometheus, including Raft-specific metrics.
- **Automatic Memory Reclamation**: Periodically rebuilds storage shards to reclaim memory from deleted items, preventing memory bloat in write-heavy workloads.

## Architecture and Design Decisions

This project was built with a focus on exploring the core principles behind modern distributed data systems.

### From Single-Node to Distributed

The store was originally a single-node application that used a traditional Write-Ahead Log (WAL) and periodic snapshots for persistence. To improve fault tolerance and scalability, it was re-architected into a distributed system using [Raft consensus algorithm](https://raft.github.io/).

### High-Throughput Concurrency

- **Problem**: A single, global lock on a central data map creates a bottleneck under concurrent loads.
- **Solution**: A **sharded map** partitions the key space across many maps, each with its own lock. This distributes write contention, significantly improving parallelism and throughput.

### Fault Tolerance and Consistency

- **Problem**: A single-node store is a single point of failure.
- **Solution**: By integrating the `raft-core` library, the key-value store is transformed into a distributed state machine. All write operations (`PUT`, `DELETE`) are submitted to the Raft log. A command is only applied to the state machine (the in-memory map) after it has been replicated to a majority of nodes in the cluster, guaranteeing strong consistency. Read operations (`GET`) are served by the leader, ensuring clients always receive up-to-date data (linearizability).

## Configuration

Before running the application, you need to create a `config.yml` file. A commented example is provided in `config/config.example.yml`. You can copy and modify it to get started:

```bash
cp config/config.example.yml config/config.yml
```

The configuration is read from the file specified by the `-cfg_path` flag or from environment variables.

## Getting Started

You can run the application as a single node or as a multi-node cluster using Docker Compose.

### With Docker (Recommended)

#### Multi-Node Cluster

To run a full 3-node cluster:

```bash
# Build and start the cluster in detached mode
@docker compose up -d --build
```

- The KV store nodes will be available at `http://localhost:8081`, `http://localhost:8082`, and `http://localhost:8083`. You can send requests to any node, and you will be redirected to the leader if necessary.
- Prometheus will be available at `http://localhost:9090`.
- Grafana will be available at `http://localhost:3000`.

### Development And Tests

Feel free to use `Makefile` for common development tasks including building, running, and linting.

## Observability

The project includes a pre-configured Grafana dashboard for visualizing performance metrics:

- **HTTP & gRPC Performance**: Request rates, latency distributions, and error rates.
- **Key-Value Operations**: P99 latency for PUT, GET, and DELETE operations.
- **Go Runtime Metrics**: Goroutines, memory usage, GC performance, etc.
- **Raft Metrics**: Raft-internal metrics are exposed on a separate port (see `docker-compose.yml`) and scraped by Prometheus.

## Performance

The application was benchmarked using k6 with an open model (arrival-rate) with fixed keys and values sizes (32 and 256 bytes respectively) on the following machine specifications:

- **CPU**: Intel Core i9-9900KF
- **RAM**: 16GB 3600MHz CL15
- **Storage**: NVMe SSD

| Metric               | Result       |
| -------------------- | ------------ |
| Achieved RPS         | ~28000 req/s |
| Failure Rate         | 0.00%        |
| p99 Latency (PUT)    | ~40ms        |
| p99 Latency (GET)    | ~8ms         |
| p99 Latency (DELETE) | ~40ms        |

**A Quick Note on Throughput**: The benchmark results presented here represent a balanced throughput, not the highest possible. It can be safely assumed that throughput of around 30,000 RPS can be sustained with approximately the same latency profile.

## Tradeoffs and Future Work

- **Dynamic Membership Options**: Implementing dynamic membership would allow nodes to join or leave the Raft cluster without manual reconfiguration and restarts, improving operational flexibility and scalability.
- **Kubernetes (k8s) Configuration**: Providing official Kubernetes manifests and deployment guides would significantly simplify the deployment and management of the KV store in containerized environments.
- **Performance Tuning**: Profile the application under load to investigate the causes of the current performance ceiling and the observed "long-tail" latency (the significant gap between p95 and max response times) to further improve performance consistency.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
