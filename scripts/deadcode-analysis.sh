#!/bin/bash
# deadcode-analysis.sh - Comprehensive deadcode analysis workflow
# 
# This script provides a complete deadcode analysis workflow including:
# - Production-only dead code detection
# - Test-only code identification
# - Cross-platform analysis
# - Detailed reporting

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEMP_DIR="$PROJECT_ROOT/.deadcode-tmp"
OUTPUT_DIR="$PROJECT_ROOT/.deadcode-reports"

# Packages to analyze
PACKAGES="./cmd/... ./internal/..."

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    if ! command -v deadcode &> /dev/null; then
        log_error "deadcode not found. Install with: go install golang.org/x/tools/cmd/deadcode@latest"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warn "jq not found. Install jq for JSON processing and test-only analysis."
        log_warn "On Ubuntu/Debian: sudo apt-get install jq"
        log_warn "On macOS: brew install jq"
        log_warn "Continuing without jq..."
        JQ_AVAILABLE=false
    else
        JQ_AVAILABLE=true
    fi
    
    log_success "Dependencies checked"
}

setup_directories() {
    log_info "Setting up directories..."
    mkdir -p "$TEMP_DIR"
    mkdir -p "$OUTPUT_DIR"
    log_success "Directories created"
}

cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"
    log_success "Cleanup complete"
}

analyze_production_deadcode() {
    log_info "Analyzing production dead code..."
    
    local output_file="$OUTPUT_DIR/deadcode-production.txt"
    deadcode $PACKAGES > "$output_file" 2>&1 || true
    
    local count=$(grep -c "unreachable func:" "$output_file" || echo "0")
    
    if [ "$count" -gt 0 ]; then
        log_warn "Found $count unreachable functions in production"
        echo "Results saved to: $output_file"
    else
        log_success "No dead code found in production"
    fi
    
    echo "$count"
}

analyze_with_tests() {
    log_info "Analyzing dead code including tests..."
    
    local output_file="$OUTPUT_DIR/deadcode-with-tests.txt"
    deadcode -test $PACKAGES > "$output_file" 2>&1 || true
    
    local count=$(grep -c "unreachable func:" "$output_file" || echo "0")
    
    if [ "$count" -gt 0 ]; then
        log_warn "Found $count unreachable functions including tests"
        echo "Results saved to: $output_file"
    else
        log_success "No dead code found including tests"
    fi
    
    echo "$count"
}

analyze_test_only_code() {
    if [ "$JQ_AVAILABLE" = false ]; then
        log_warn "Skipping test-only analysis (jq not available)"
        return
    fi
    
    log_info "Analyzing test-only code..."
    
    # Generate JSON reports
    deadcode -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$TEMP_DIR/dead_prod.txt"
    deadcode -test -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$TEMP_DIR/dead_with_tests.txt"
    
    # Find test-only functions
    local output_file="$OUTPUT_DIR/test-only-functions.txt"
    comm -23 "$TEMP_DIR/dead_prod.txt" "$TEMP_DIR/dead_with_tests.txt" > "$output_file" || true
    
    local count=$(wc -l < "$output_file" || echo "0")
    
    if [ "$count" -gt 0 ]; then
        log_warn "Found $count functions used only by tests"
        echo "Test-only functions:"
        cat "$output_file"
        echo "Results saved to: $output_file"
    else
        log_success "No test-only functions found"
    fi
    
    echo "$count"
}

analyze_cross_platform() {
    log_info "Analyzing cross-platform dead code..."
    
    local platforms=("linux/amd64" "darwin/amd64" "windows/amd64")
    local all_dead_functions="$TEMP_DIR/all_dead_functions.txt"
    local common_dead_functions="$OUTPUT_DIR/common-dead-functions.txt"
    
    # Initialize with first platform
    GOOS=$(echo "${platforms[0]}" | cut -d'/' -f1)
    GOARCH=$(echo "${platforms[0]}" | cut -d'/' -f2)
    log_info "Analyzing $GOOS/$GOARCH..."
    
    deadcode -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$all_dead_functions"
    
    # Intersect with other platforms
    for platform in "${platforms[@]:1}"; do
        GOOS=$(echo "$platform" | cut -d'/' -f1)
        GOARCH=$(echo "$platform" | cut -d'/' -f2)
        log_info "Analyzing $GOOS/$GOARCH..."
        
        local platform_dead="$TEMP_DIR/dead_${GOOS}_${GOARCH}.txt"
        deadcode -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$platform_dead"
        
        # Intersect with previous results
        comm -12 "$all_dead_functions" "$platform_dead" > "$TEMP_DIR/temp_intersection.txt"
        mv "$TEMP_DIR/temp_intersection.txt" "$all_dead_functions"
    done
    
    mv "$all_dead_functions" "$common_dead_functions"
    
    local count=$(wc -l < "$common_dead_functions" || echo "0")
    
    if [ "$count" -gt 0 ]; then
        log_warn "Found $count functions dead across all platforms"
        echo "Cross-platform dead functions:"
        cat "$common_dead_functions"
        echo "Results saved to: $common_dead_functions"
    else
        log_success "No functions are dead across all platforms"
    fi
    
    echo "$count"
}

generate_summary_report() {
    log_info "Generating summary report..."
    
    local summary_file="$OUTPUT_DIR/deadcode-summary.md"
    
    cat > "$summary_file" << EOF
# Deadcode Analysis Summary

Generated on: $(date)
Project: Spin
Packages analyzed: $PACKAGES

## Results

### Production Dead Code
- **Count**: $(analyze_production_deadcode)
- **File**: deadcode-production.txt

### Dead Code Including Tests  
- **Count**: $(analyze_with_tests)
- **File**: deadcode-with-tests.txt

### Test-Only Functions
- **Count**: $(analyze_test_only_code)
- **File**: test-only-functions.txt

### Cross-Platform Dead Code
- **Count**: $(analyze_cross_platform)
- **File**: common-dead-functions.txt

## Recommendations

1. **Review production dead code** - These functions are not reachable from main packages
2. **Consider test-only functions** - These may indicate gaps in production usage
3. **Cross-platform analysis** - Functions dead across all platforms are safe candidates for removal
4. **Use deadcode-why** - Investigate specific functions with \`deadcode -whylive <function>\`

## Usage

\`\`\`bash
# Run full analysis
./scripts/deadcode-analysis.sh

# Check specific function
deadcode -whylive "functionName" ./cmd/... ./internal/...

# Production only
deadcode ./cmd/... ./internal/...

# With tests
deadcode -test ./cmd/... ./internal/...
\`\`\`

## Notes

- Always review findings before removing code
- Consider interface implementations and reflection usage
- Some functions may be used by external tools or future features
- Generated code is excluded by default (use -generated flag to include)
EOF

    log_success "Summary report generated: $summary_file"
}

show_help() {
    cat << EOF
deadcode-analysis.sh - Comprehensive deadcode analysis workflow

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help          Show this help message
    -p, --production    Analyze production dead code only
    -t, --test-only     Find test-only functions only
    -c, --cross-platform Analyze cross-platform dead code only
    -s, --summary       Generate summary report only
    -a, --all           Run all analyses (default)
    --clean             Clean output directories and exit

EXAMPLES:
    $0                  # Run all analyses
    $0 --production     # Production dead code only
    $0 --test-only      # Test-only functions only
    $0 --summary        # Generate summary report

REQUIREMENTS:
    - deadcode: go install golang.org/x/tools/cmd/deadcode@latest
    - jq: for JSON processing (optional but recommended)

OUTPUT:
    Reports are saved to .deadcode-reports/ directory
EOF
}

main() {
    local run_all=true
    local run_production=false
    local run_test_only=false
    local run_cross_platform=false
    local run_summary=false
    local clean_only=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -p|--production)
                run_all=false
                run_production=true
                shift
                ;;
            -t|--test-only)
                run_all=false
                run_test_only=true
                shift
                ;;
            -c|--cross-platform)
                run_all=false
                run_cross_platform=true
                shift
                ;;
            -s|--summary)
                run_all=false
                run_summary=true
                shift
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            --clean)
                clean_only=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    if [ "$clean_only" = true ]; then
        rm -rf "$OUTPUT_DIR" "$TEMP_DIR"
        log_success "Cleaned output directories"
        exit 0
    fi
    
    log_info "Starting deadcode analysis..."
    
    check_dependencies
    setup_directories
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    if [ "$run_all" = true ] || [ "$run_production" = true ]; then
        analyze_production_deadcode
    fi
    
    if [ "$run_all" = true ] || [ "$run_test_only" = true ]; then
        analyze_test_only_code
    fi
    
    if [ "$run_all" = true ] || [ "$run_cross_platform" = true ]; then
        analyze_cross_platform
    fi
    
    if [ "$run_all" = true ] || [ "$run_summary" = true ]; then
        generate_summary_report
    fi
    
    log_success "Deadcode analysis complete!"
    log_info "Reports saved to: $OUTPUT_DIR"
    
    if [ "$run_all" = true ]; then
        echo ""
        echo "Quick commands:"
        echo "  make deadcode-prod      # Production dead code"
        echo "  make deadcode-test-only # Test-only functions"
        echo "  make deadcode-why FUNC=functionName # Why function is not dead"
    fi
}

# Run main function with all arguments
main "$@"
