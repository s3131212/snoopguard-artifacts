#!/bin/bash

# Output directory for all results
OUTPUT_DIR="benchmark_results"
mkdir -p "$OUTPUT_DIR"

# CPU step range
CPU_START=0.5
CPU_END=1.0
CPU_STEP=0.1

# Loop over different CPU values
current_cpu="$CPU_START"
while (( $(echo "$current_cpu <= $CPU_END" | bc -l) )); do
    echo "Running benchmarks with --cpus=$current_cpu..."

    # Format current_cpu to ensure leading zero
    formatted_cpu=$(printf "%.1f" "$current_cpu")

    # Create a directory for this CPU configuration
    CPU_DIR="$OUTPUT_DIR/cpu_$formatted_cpu"
    mkdir -p "$CPU_DIR"

    # Run Geekbench with specified CPUs
    echo "Running Geekbench..."
    geekbench_output_file="$CPU_DIR/geekbench_output.txt"
    docker run --rm --cpus="$formatted_cpu" geekbench | tee "$geekbench_output_file"

    # Run the benchmark script
    echo "Running test script..."
    docker run --rm --cpus="$formatted_cpu" -v "$(pwd)/$CPU_DIR:/app/benchmark_results/native" go-app

    # Increment CPU step
    current_cpu=$(echo "$current_cpu + $CPU_STEP" | bc -l)
    echo ""
done

echo "All benchmarks completed. Results are in $OUTPUT_DIR."
