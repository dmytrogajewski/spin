#!/bin/bash
# deadcode-ci.sh - CI-friendly deadcode analysis
# 
# This script is designed for CI environments and provides:
# - Exit codes for CI pipeline integration
# - Configurable thresholds
# - JSON output for programmatic processing
# - Minimal output for clean CI logs

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PACKAGES="./cmd/... ./internal/..."

# Default thresholds
MAX_DEAD_CODE_PROD=50
MAX_DEAD_CODE_WITH_TESTS=100
MAX_TEST_ONLY_FUNCTIONS=20

# Colors for output (only if not in CI)
if [ "${CI:-false}" = "false" ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

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
    if ! command -v deadcode &> /dev/null; then
        log_error "deadcode not found. Install with: go install golang.org/x/tools/cmd/deadcode@latest"
        exit 1
    fi
}

count_dead_functions() {
    local output
    output=$(deadcode "$@" 2>&1 || true)
    echo "$output" | grep -c "unreachable func:" || echo "0"
}

analyze_production() {
    log_info "Analyzing production dead code..."
    local count
    count=$(count_dead_functions $PACKAGES)
    
    if [ "$count" -gt "$MAX_DEAD_CODE_PROD" ]; then
        log_error "Production dead code exceeds threshold: $count > $MAX_DEAD_CODE_PROD"
        return 1
    elif [ "$count" -gt 0 ]; then
        log_warn "Production dead code found: $count functions"
    else
        log_success "No production dead code found"
    fi
    
    echo "PROD_DEAD_CODE_COUNT=$count" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
    return 0
}

analyze_with_tests() {
    log_info "Analyzing dead code including tests..."
    local count
    count=$(count_dead_functions -test $PACKAGES)
    
    if [ "$count" -gt "$MAX_DEAD_CODE_WITH_TESTS" ]; then
        log_error "Dead code with tests exceeds threshold: $count > $MAX_DEAD_CODE_WITH_TESTS"
        return 1
    elif [ "$count" -gt 0 ]; then
        log_warn "Dead code with tests found: $count functions"
    else
        log_success "No dead code found including tests"
    fi
    
    echo "WITH_TESTS_DEAD_CODE_COUNT=$count" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
    return 0
}

analyze_test_only() {
    if ! command -v jq &> /dev/null; then
        log_warn "jq not available, skipping test-only analysis"
        return 0
    fi
    
    log_info "Analyzing test-only functions..."
    
    local temp_dir
    temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT
    
    # Generate function lists
    deadcode -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$temp_dir/dead_prod.txt"
    deadcode -test -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$temp_dir/dead_with_tests.txt"
    
    # Find test-only functions
    local count
    count=$(comm -23 "$temp_dir/dead_prod.txt" "$temp_dir/dead_with_tests.txt" | wc -l || echo "0")
    
    if [ "$count" -gt "$MAX_TEST_ONLY_FUNCTIONS" ]; then
        log_error "Test-only functions exceed threshold: $count > $MAX_TEST_ONLY_FUNCTIONS"
        return 1
    elif [ "$count" -gt 0 ]; then
        log_warn "Test-only functions found: $count functions"
    else
        log_success "No test-only functions found"
    fi
    
    echo "TEST_ONLY_FUNCTIONS_COUNT=$count" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
    return 0
}

generate_json_report() {
    log_info "Generating JSON report..."
    
    local report_file="${CI_PROJECT_DIR:-$PROJECT_ROOT}/deadcode-report.json"
    
    cat > "$report_file" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "project": "spin",
  "packages": "$PACKAGES",
  "thresholds": {
    "max_production_dead_code": $MAX_DEAD_CODE_PROD,
    "max_dead_code_with_tests": $MAX_DEAD_CODE_WITH_TESTS,
    "max_test_only_functions": $MAX_TEST_ONLY_FUNCTIONS
  },
  "results": {
    "production_dead_code_count": $(count_dead_functions $PACKAGES),
    "dead_code_with_tests_count": $(count_dead_functions -test $PACKAGES),
    "test_only_functions_count": $(if command -v jq &> /dev/null; then
        local temp_dir=$(mktemp -d)
        deadcode -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$temp_dir/dead_prod.txt"
        deadcode -test -json $PACKAGES | jq -r '.[] | .Funcs[].Name' | sort > "$temp_dir/dead_with_tests.txt"
        comm -23 "$temp_dir/dead_prod.txt" "$temp_dir/dead_with_tests.txt" | wc -l || echo "0"
        rm -rf "$temp_dir"
    else
        echo "0"
    fi)
  }
}
EOF
    
    log_success "JSON report generated: $report_file"
}

show_help() {
    cat << EOF
deadcode-ci.sh - CI-friendly deadcode analysis

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -p, --production         Check production dead code only
    -t, --test-only         Check test-only functions only
    -w, --with-tests        Check dead code with tests only
    -j, --json              Generate JSON report only
    -a, --all               Run all checks (default)
    --max-prod N            Set max production dead code threshold (default: $MAX_DEAD_CODE_PROD)
    --max-tests N           Set max dead code with tests threshold (default: $MAX_DEAD_CODE_WITH_TESTS)
    --max-test-only N       Set max test-only functions threshold (default: $MAX_TEST_ONLY_FUNCTIONS)

EXAMPLES:
    $0                      # Run all checks with default thresholds
    $0 --max-prod 100      # Set custom production threshold
    $0 --production         # Check production dead code only
    $0 --json               # Generate JSON report only

EXIT CODES:
    0 - All checks passed
    1 - One or more checks failed (exceeded thresholds)
    2 - Error in script execution

ENVIRONMENT VARIABLES:
    CI                      Set to 'true' for CI mode (reduces colored output)
    GITHUB_OUTPUT           GitHub Actions output file (if available)
    CI_PROJECT_DIR          Project directory for reports (if available)
EOF
}

main() {
    local run_all=true
    local run_production=false
    local run_test_only=false
    local run_with_tests=false
    local run_json=false
    local exit_code=0
    
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
            -w|--with-tests)
                run_all=false
                run_with_tests=true
                shift
                ;;
            -j|--json)
                run_all=false
                run_json=true
                shift
                ;;
            -a|--all)
                run_all=true
                shift
                ;;
            --max-prod)
                MAX_DEAD_CODE_PROD="$2"
                shift 2
                ;;
            --max-tests)
                MAX_DEAD_CODE_WITH_TESTS="$2"
                shift 2
                ;;
            --max-test-only)
                MAX_TEST_ONLY_FUNCTIONS="$2"
                shift 2
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
    
    log_info "Starting CI deadcode analysis..."
    log_info "Thresholds: prod=$MAX_DEAD_CODE_PROD, with_tests=$MAX_DEAD_CODE_WITH_TESTS, test_only=$MAX_TEST_ONLY_FUNCTIONS"
    
    check_dependencies
    
    if [ "$run_all" = true ] || [ "$run_production" = true ]; then
        if ! analyze_production; then
            exit_code=1
        fi
    fi
    
    if [ "$run_all" = true ] || [ "$run_with_tests" = true ]; then
        if ! analyze_with_tests; then
            exit_code=1
        fi
    fi
    
    if [ "$run_all" = true ] || [ "$run_test_only" = true ]; then
        if ! analyze_test_only; then
            exit_code=1
        fi
    fi
    
    if [ "$run_all" = true ] || [ "$run_json" = true ]; then
        generate_json_report
    fi
    
    if [ "$exit_code" -eq 0 ]; then
        log_success "All deadcode checks passed!"
    else
        log_error "One or more deadcode checks failed!"
    fi
    
    exit $exit_code
}

# Run main function with all arguments
main "$@"
