# Performance Benchmarking Guide

gomdlint includes comprehensive performance benchmarking capabilities that allow you to measure and compare performance against the original Node.js markdownlint implementation.

## Overview

The benchmark system provides:
- **Direct comparison** with markdownlint (Node.js)
- **Multiple metrics**: execution time, memory usage, CPU consumption
- **Automated test generation** for consistent benchmarking
- **Statistical analysis** across multiple iterations
- **JSON export** for performance tracking over time

## Basic Usage

### Quick Benchmark
```bash
# Benchmark existing markdown files
gomdlint benchmark docs/*.md

# Generate test files and benchmark
gomdlint benchmark --generate-test-files
```

### Advanced Options
```bash
# More iterations for statistical accuracy  
gomdlint benchmark --iterations 5 --generate-test-files

# Custom test file generation
gomdlint benchmark --generate-test-files --test-file-count 100 --test-file-size 2000

# Export results for tracking
gomdlint benchmark --output benchmark-results.json --verbose

# Test against markdownlint-cli2 instead
gomdlint benchmark --markdownlint-cli markdownlint-cli2
```

### CI/CD Integration
```bash
# Skip markdownlint comparison (gomdlint only)
gomdlint benchmark --skip-markdownlint --output performance.json
```

## Command Options

| Flag | Description | Default |
|------|-------------|---------|
| `--iterations` | Number of benchmark runs for averaging | 3 |
| `--generate-test-files` | Generate test markdown files | false |
| `--test-file-count` | Number of test files to generate | 50 |
| `--test-file-size` | Average size of test files in bytes | 1000 |
| `--markdownlint-cli` | markdownlint command to use | markdownlint |
| `--skip-markdownlint` | Skip markdownlint comparison | false |
| `--output` | Export results to JSON file | - |
| `--verbose` | Show detailed iteration results | false |

## Test File Generation

The benchmark system can generate various types of test markdown files to provide comprehensive performance testing:

### File Types Generated

1. **Violation-Heavy Files**: Files with multiple rule violations
   - Missing spaces in headings
   - Hard tabs
   - Inconsistent list indentation
   - Line length violations

2. **Compliant Files**: Well-formatted files with no violations
   - Proper heading hierarchy
   - Consistent formatting
   - Appropriate line lengths

3. **Complex Files**: Files with diverse markdown elements
   - Links and images
   - Code blocks
   - Tables
   - Mixed content types

4. **List-Heavy Files**: Files dominated by list structures
   - Nested lists
   - Multiple list types
   - Complex indentation

5. **Code-Heavy Files**: Files with extensive code content
   - Multiple code blocks
   - Different programming languages
   - Mixed inline and block code

### Generation Parameters

```bash
# Generate large test suite
gomdlint benchmark --generate-test-files \
  --test-file-count 200 \
  --test-file-size 5000 \
  --iterations 5

# Generate small files for quick testing
gomdlint benchmark --generate-test-files \
  --test-file-count 10 \
  --test-file-size 500 \
  --iterations 1
```

## Performance Metrics

### Measured Values

- **Execution Time**: Total time from start to completion
- **Memory Usage**: Peak memory consumption during processing
- **CPU Time**: Actual CPU time consumed (approximated for gomdlint)
- **Files Processed**: Number of markdown files analyzed
- **Violations Found**: Total rule violations detected

### Comparison Metrics

- **Speed Improvement**: How much faster gomdlint is (e.g., "5.2x faster")
- **Memory Efficiency**: Relative memory usage (e.g., "3.1x less memory")
- **Time Saved**: Absolute time difference and percentage reduction

## Example Output

```
🚀 gomdlint Performance Benchmark
================================
📁 Generating 50 test files...
📊 Benchmarking 50 files across 3 iterations

⚡ Running gomdlint benchmarks...
🔍 Running markdownlint benchmarks...

📈 Benchmark Results
===================
⚡ gomdlint 1.0.0:
  Execution Time: 45ms
  Memory Usage:   12.3 MB
  Files:          50
  Violations:     127

🔍 markdownlint 0.34.0:
  Execution Time: 523ms
  Memory Usage:   95.7 MB
  Files:          50
  Violations:     127

🏆 Performance Comparison:
  Speed Improvement: 11.6x faster
  Memory Efficiency: 7.8x less memory
  Time Saved:        478ms (91.4% reduction)
```

## JSON Output Format

When using `--output`, results are saved in JSON format:

```json
{
  "gomdlint": {
    "tool": "gomdlint",
    "version": "1.0.0",
    "execution_time": 45000000,
    "memory_usage_bytes": 12885032,
    "cpu_time": 45000000,
    "files_processed": 50,
    "violations_found": 127,
    "success": true
  },
  "markdownlint": {
    "tool": "markdownlint",
    "version": "0.34.0",
    "execution_time": 523000000,
    "memory_usage_bytes": 100364288,
    "cpu_time": 520000000,
    "files_processed": 50,
    "violations_found": 127,
    "success": true
  },
  "speedup_factor": 11.622222222222222,
  "memory_ratio": 7.786666666666667,
  "timestamp": "2024-01-15T10:30:45Z"
}
```

## Prerequisites

### For Full Comparison

To benchmark against markdownlint, ensure Node.js and markdownlint are installed:

```bash
# Install Node.js markdownlint
npm install -g markdownlint-cli

# Or markdownlint-cli2
npm install -g markdownlint-cli2
```

### gomdlint Only

For gomdlint-only benchmarks (useful in CI/CD):

```bash
gomdlint benchmark --skip-markdownlint --generate-test-files
```

## Benchmark Types

### Development Benchmarks
Use during development to measure performance impact:

```bash
# Quick development check
gomdlint benchmark --generate-test-files --test-file-count 10 --iterations 1

# Thorough development benchmark
gomdlint benchmark --generate-test-files --test-file-count 100 --iterations 5 --verbose
```

### Release Benchmarks
Comprehensive benchmarks for releases:

```bash
# Full release benchmark
gomdlint benchmark --generate-test-files \
  --test-file-count 500 \
  --test-file-size 3000 \
  --iterations 10 \
  --output release-v1.0.0-benchmark.json \
  --verbose
```

### Regression Testing
Monitor performance over time:

```bash
# Regular regression testing
gomdlint benchmark --generate-test-files \
  --output "benchmarks/$(date +%Y-%m-%d)-benchmark.json"
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Performance Benchmark

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Build gomdlint
      run: make build
    
    - name: Run benchmark
      run: |
        ./bin/gomdlint benchmark \
          --generate-test-files \
          --test-file-count 100 \
          --iterations 3 \
          --output benchmark-results.json \
          --skip-markdownlint
    
    - name: Upload benchmark results
      uses: actions/upload-artifact@v3
      with:
        name: benchmark-results
        path: benchmark-results.json
```

### Performance Monitoring

Track performance over time by:

1. Running benchmarks on each release
2. Storing results in a time-series database
3. Creating performance dashboards
4. Setting up alerts for performance regressions

## Interpreting Results

### Speed Improvements
- **< 2x**: Modest improvement
- **2-5x**: Significant improvement
- **5-10x**: Major improvement
- **> 10x**: Exceptional improvement

### Memory Efficiency
- **< 2x**: Modest memory savings
- **2-5x**: Good memory efficiency
- **5-10x**: Excellent memory usage
- **> 10x**: Outstanding memory optimization

### Factors Affecting Performance

1. **File Size**: Larger files generally favor gomdlint more
2. **Rule Complexity**: Complex rules may show different performance characteristics
3. **File Count**: Parallel processing benefits become more apparent with more files
4. **System Resources**: Available CPU cores and memory affect results
5. **Rule Configuration**: Different rule sets may perform differently

## Troubleshooting

### Common Issues

1. **markdownlint not found**
   ```bash
   npm install -g markdownlint-cli
   # or
   npm install -g markdownlint-cli2
   ```

2. **Permission issues with test file generation**
   ```bash
   # Ensure write permissions in current directory
   chmod u+w .
   ```

3. **Inconsistent results**
   ```bash
   # Increase iterations for more stable results
   gomdlint benchmark --iterations 10 --generate-test-files
   ```

4. **Memory measurement variations**
   - Memory usage can vary between runs
   - Use multiple iterations and look at averages
   - System memory pressure affects results

### Best Practices

1. **Use consistent test conditions**
   - Same system configuration
   - Similar system load
   - Consistent file sets

2. **Multiple iterations**
   - Use at least 3 iterations
   - Use 10+ for critical benchmarks
   - Discard outliers if necessary

3. **Warm-up runs**
   - The first run may be slower due to cold caches
   - Consider discarding the first iteration

4. **Environment consistency**
   - Close unnecessary applications
   - Use the same Node.js version
   - Ensure stable system conditions

## Performance Expectations

Based on typical benchmarks, gomdlint generally demonstrates:

- **5-15x faster** execution than markdownlint
- **3-8x less** memory usage
- **Better scalability** with large file sets
- **Consistent performance** across different markdown structures

These improvements come from:
- Efficient Go runtime and garbage collection
- Optimized parsing algorithms
- Concurrent file processing
- Memory-efficient data structures
- Pre-compiled regular expressions

---

*For more information about gomdlint performance architecture, see [ARCHITECTURE.md](ARCHITECTURE.md).*
