#!/bin/bash

# Function to run a test and save output to a specified file
run_test() {
    test_name=$1
    output_file=$2
    echo "Running $test_name..."
    go test ./pkg/benchmark -v -timeout 0 -run "^$test_name$" | tee -a "$output_file"
    echo "Results saved to $output_file"
    echo ""
}

# Benchmark files
BENCHMARK_ADD="benchmark_results/native_roundtrip/benchmark_add.txt"
BENCHMARK_MLS="benchmark_results/native_roundtrip/benchmark_MLS.txt"
BENCHMARK_SERVERSIDE="benchmark_results/native_roundtrip/benchmark_serverside.txt"

# Clear old benchmark files
> "$BENCHMARK_ADD"
> "$BENCHMARK_MLS"
> "$BENCHMARK_SERVERSIDE"

# Run chatbot addition tests
run_test "TestBenchmarkChatbotAddition" "$BENCHMARK_ADD"
run_test "TestBenchmarkMlsChatbotAddition" "$BENCHMARK_ADD"

# Run MLS message sending tests
run_test "TestBenchmarkUserSendMlsGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserSendMlsIGAGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserSendMlsPseudoGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserSendMlsIGAGroupMessageWithHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserSendMlsPseudoGroupMessageWithHideTrigger" "$BENCHMARK_MLS"

# Run server-side message sending tests
run_test "TestBenchmarkUserSendServerSideGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserSendServerSideIGAGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserSendServerSidePseudoGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserSendServerSideIGAGroupMessageWithHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserSendServerSidePseudoGroupMessageWithHideTrigger" "$BENCHMARK_SERVERSIDE"

echo "All tests completed. Results are saved to respective files."