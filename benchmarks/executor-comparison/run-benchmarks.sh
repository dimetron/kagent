#!/bin/bash

# Executor Performance Benchmark Runner
# Compares Go Temporal executor vs Python executors

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "  Executor Performance Benchmarks"
echo "=========================================="
echo ""

# Configuration
ITERATIONS=${ITERATIONS:-1000}
WARMUP=${WARMUP:-100}
SCENARIO=${1:-all}
OUTPUT_DIR="results/$(date +%Y%m%d-%H%M%S)"

mkdir -p "$OUTPUT_DIR"

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."

    # Check Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go not found${NC}"
        exit 1
    fi

    # Check Python
    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}Error: Python not found${NC}"
        exit 1
    fi

    # Check Docker (for Temporal)
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: Docker not found${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ All prerequisites met${NC}"
    echo ""
}

# Start Temporal
start_temporal() {
    echo "Starting Temporal..."
    cd ../../deployments/temporal-executor
    docker-compose up -d
    cd -

    # Wait for Temporal to be ready
    echo "Waiting for Temporal to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:7233 > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Temporal is ready${NC}"
            return 0
        fi
        sleep 1
    done

    echo -e "${RED}Error: Temporal failed to start${NC}"
    exit 1
}

# Run Go Temporal benchmarks
run_go_benchmarks() {
    echo ""
    echo "Running Go Temporal benchmarks..."
    echo "=================================="

    cd ../../go/internal/executor/temporal

    # Run benchmarks
    go test -bench=. -benchmem -benchtime=${ITERATIONS}x \
        -cpuprofile="$OUTPUT_DIR/go-cpu.prof" \
        -memprofile="$OUTPUT_DIR/go-mem.prof" \
        | tee "$OUTPUT_DIR/go-results.txt"

    cd -

    echo -e "${GREEN}✓ Go benchmarks complete${NC}"
}

# Run Python benchmarks
run_python_benchmarks() {
    echo ""
    echo "Running Python benchmarks..."
    echo "============================"

    # LangGraph
    echo "Testing LangGraph executor..."
    python3 benchmark-python.py \
        --executor langgraph \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/python-langgraph.json"

    # ADK
    echo "Testing ADK executor..."
    python3 benchmark-python.py \
        --executor adk \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/python-adk.json"

    # CrewAI
    echo "Testing CrewAI executor..."
    python3 benchmark-python.py \
        --executor crewai \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/python-crewai.json"

    echo -e "${GREEN}✓ Python benchmarks complete${NC}"
}

# Run specific scenario
run_scenario() {
    local scenario=$1
    echo ""
    echo "Running scenario: $scenario"
    echo "=============================="

    case $scenario in
        simple-chat)
            run_simple_chat_benchmark
            ;;
        tool-execution)
            run_tool_execution_benchmark
            ;;
        multi-turn)
            run_multi_turn_benchmark
            ;;
        complex-loop)
            run_complex_loop_benchmark
            ;;
        concurrent)
            run_concurrent_benchmark
            ;;
        *)
            echo "Unknown scenario: $scenario"
            exit 1
            ;;
    esac
}

# Simple chat benchmark
run_simple_chat_benchmark() {
    echo "Scenario: Simple Chat (no tools)"
    python3 benchmark-scenario.py \
        --scenario simple-chat \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/scenario-simple-chat.json"
}

# Tool execution benchmark
run_tool_execution_benchmark() {
    echo "Scenario: Tool Execution"
    python3 benchmark-scenario.py \
        --scenario tool-execution \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/scenario-tool-execution.json"
}

# Multi-turn benchmark
run_multi_turn_benchmark() {
    echo "Scenario: Multi-Turn Conversation"
    python3 benchmark-scenario.py \
        --scenario multi-turn \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/scenario-multi-turn.json"
}

# Complex loop benchmark
run_complex_loop_benchmark() {
    echo "Scenario: Complex Agent Loop"
    python3 benchmark-scenario.py \
        --scenario complex-loop \
        --iterations $ITERATIONS \
        --output "$OUTPUT_DIR/scenario-complex-loop.json"
}

# Concurrent benchmark
run_concurrent_benchmark() {
    echo "Scenario: Concurrent Requests"
    python3 benchmark-scenario.py \
        --scenario concurrent \
        --concurrent 100 \
        --iterations 100 \
        --output "$OUTPUT_DIR/scenario-concurrent.json"
}

# Generate comparison report
generate_report() {
    echo ""
    echo "Generating comparison report..."
    echo "==============================="

    python3 compare-results.py \
        --input-dir "$OUTPUT_DIR" \
        --output "$OUTPUT_DIR/comparison-report.md"

    echo -e "${GREEN}✓ Report generated: $OUTPUT_DIR/comparison-report.md${NC}"
    echo ""
    echo "Summary:"
    cat "$OUTPUT_DIR/comparison-report.md"
}

# Cleanup
cleanup() {
    echo ""
    echo "Cleaning up..."
    cd ../../deployments/temporal-executor
    docker-compose down
    cd -
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

# Main execution
main() {
    check_prerequisites
    start_temporal

    if [ "$SCENARIO" == "all" ]; then
        run_go_benchmarks
        run_python_benchmarks

        # Run all scenarios
        for scenario in simple-chat tool-execution multi-turn complex-loop concurrent; do
            run_scenario $scenario
        done
    else
        run_scenario $SCENARIO
    fi

    generate_report

    # Cleanup if not in CI
    if [ -z "$CI" ]; then
        cleanup
    fi

    echo ""
    echo -e "${GREEN}=========================================="
    echo "  Benchmarks Complete!"
    echo "==========================================${NC}"
    echo ""
    echo "Results saved to: $OUTPUT_DIR"
}

# Handle Ctrl+C
trap cleanup EXIT

# Run
main
