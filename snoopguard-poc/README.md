# SnoopGuard Reference Implementation

This repository contains the reference implementation of **SnoopGuard**, a privacy-preserving group messaging protocol designed to protect users from overprivileged chatbots in group chats. Our design addresses privacy concerns through **Selective Message Access** and **Sender Anonymity**, while maintaining end-to-end encryption (E2EE) and performance efficiency.

## Repository Structure

### CMRT (Compressed Multi-Roots Tree) Implementation
- **`pkg/treekem`**: Implements TreeKEM and CMRT to enable efficient group key management.
- **`pkg/mls_multi_tree`**: Adapts CMRT to utilize the Message Layer Security (MLS) tree structure for user subtrees.

### Messaging Protocol Implementation
- **`pkg/user`**: User implementation.
  - **`server_side_group.go`**: SnoopGuard implementation based on the Sender Keys Protocol.
  - **`mls_group.go`**: SnoopGuard implementation based on the MLS protocol.
  - **`user_test.go`**: Unit tests for group chat protocol without chatbot interaction.
- **`pkg/chatbot`**: Chatbot implementation.
  - **`server_side_group.go`**: SnoopGuard implementation for chatbots using the Sender Keys Protocol.
  - **`mls_group.go`**: SnoopGuard implementation for chatbots using the MLS protocol.
  - **`chatbot_test.go`**: Unit tests for the SnoopGuard chatbot integration.

### Benchmarking
- **`pkg/benchmark/benchmark_test.go`**: Benchmark tests for SnoopGuard.
- **`benchmark_results/`**: Contains benchmark output files.
- **`benchmark_analysis/`**: Scripts for exporting benchmark results.
- **`benchmark.sh`**: Runs benchmarks in resource-constrained Docker containers (see Appendix D of the paper).
- **`benchmark_roundtrip.sh`**: Runs benchmarks as outlined in Section 6.2.2 of the paper.
  - Refer to the paper for differences between these two benchmarks.
- **`geekbench/`**: Dockerfile for creating a Docker image to run Geekbench benchmarks.

## Running Tests
To execute the test cases for the user and chatbot implementations:
```bash
go test ./pkg/user -v -timeout 0
go test ./pkg/chatbot -v -timeout 0
```

## Running Benchmarks

### Individual Benchmark
Evaluates key generation and message encryption for a single user.

```
sh benchmark.sh
```

Results are saved in the `./benchmark_results/native/` directory. The following files will be generated:
- `benchmark_add.txt`: Chatbot addition for both MLS and Sender Keys Protocol (marked as "server-side").
- `benchmark_MLS.txt`: MLS messaging.
- `benchmark_serverside.txt`: Sender Keys Protocol messaging.

### Roundtrip Benchmark
Measures the end-to-end completion time for user addition processes or message sending across all users. Detailed experiment methodology and results can be found in Section 6.2.2 of the paper.

```
sh benchmark_roundtrip.sh
```

Results are saved in the `./benchmark_results/native_roundtrip/` directory.

### Resource-Constrained Benchmark
Measures key generation and message encryption for a single user in a resource-constrained environment (Docker containers with limited CPU resources). For detailed experiment methodology and results, refer to Appendix D of the paper. Note: This script may take several days to complete.

Build the necessary Docker images:
```
docker build -t go-app .
docker build -t geekbench ./geekbench
```

Execute the benchmark script:
```
sh benchmark_resource_constrained.sh # modify this script to adjust resource limitations and toggle the execution of Geekbench.
```

Results are saved in the `./benchmark_results/cpu_*` directory.