#!/bin/bash
# Script to detect the number of processors/cores available
# Based on GEICO basset project's approach with enhanced error handling

set -euo pipefail

# Try different methods to get processor count based on OS
get_processor_count() {
    # Linux - try nproc first (most reliable)
    if command -v nproc >/dev/null 2>&1; then
        nproc
        return 0
    fi
    
    # macOS - use sysctl
    if command -v sysctl >/dev/null 2>&1; then
        sysctl -n hw.ncpu 2>/dev/null || sysctl -n hw.logicalcpu 2>/dev/null || echo ""
        return 0
    fi
    
    # Fallback - check /proc/cpuinfo on Linux
    if [ -r /proc/cpuinfo ]; then
        grep -c ^processor /proc/cpuinfo 2>/dev/null || echo ""
        return 0
    fi
    
    # Windows with PowerShell (if running in WSL or Git Bash)
    if command -v powershell.exe >/dev/null 2>&1; then
        powershell.exe -Command "(Get-CimInstance -ClassName Win32_Processor | Measure-Object -Property NumberOfLogicalProcessors -Sum).Sum" 2>/dev/null || echo ""
        return 0
    fi
    
    # Last resort - environment variable
    if [ -n "${NUMBER_OF_PROCESSORS:-}" ]; then
        echo "$NUMBER_OF_PROCESSORS"
        return 0
    fi
    
    # If all else fails, return empty (caller should handle fallback)
    echo ""
}

# Get the processor count
PROC_COUNT=$(get_processor_count)

# Validate the result is a positive integer
if [[ "$PROC_COUNT" =~ ^[1-9][0-9]*$ ]]; then
    echo "$PROC_COUNT"
else
    # Return empty if invalid (caller handles fallback to 1)
    echo ""
fi
