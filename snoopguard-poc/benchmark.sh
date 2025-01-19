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
BENCHMARK_ADD="benchmark_results/native/benchmark_add.txt"
BENCHMARK_MLS="benchmark_results/native/benchmark_MLS.txt"
BENCHMARK_SERVERSIDE="benchmark_results/native/benchmark_serverside.txt"

# Clear old benchmark files
> "$BENCHMARK_ADD"
> "$BENCHMARK_MLS"
> "$BENCHMARK_SERVERSIDE"

# Run chatbot addition tests
run_test "TestBenchmarkChatbotAdditionSingleUser" "$BENCHMARK_ADD"
run_test "TestBenchmarkMlsChatbotAdditionSingleUser" "$BENCHMARK_ADD"

# Run MLS message generation tests
run_test "TestBenchmarkUserGenerateMlsGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserGenerateMlsIGAGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserGenerateMlsPseudoGroupMessageWithoutHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserGenerateMlsIGAGroupMessageWithHideTrigger" "$BENCHMARK_MLS"
run_test "TestBenchmarkUserGenerateMlsPseudoGroupMessageWithHideTrigger" "$BENCHMARK_MLS"

# Run server-side message generation tests
run_test "TestBenchmarkUserGenerateServerSideGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserGenerateServerSideIGAGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserGenerateServerSidePseudoGroupMessageWithoutHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserGenerateServerSideIGAGroupMessageWithHideTrigger" "$BENCHMARK_SERVERSIDE"
run_test "TestBenchmarkUserGenerateServerSidePseudoGroupMessageWithHideTrigger" "$BENCHMARK_SERVERSIDE"

echo "All tests completed. Results are saved to respective files."