package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/interfaces/cli/output"
	"github.com/gomdlint/gomdlint/pkg/gomdlint"
	"github.com/spf13/cobra"
)

// BenchmarkResult represents the results of a benchmark run
type BenchmarkResult struct {
	Tool            string        `json:"tool"`
	Version         string        `json:"version"`
	ExecutionTime   time.Duration `json:"execution_time"`
	MemoryUsage     int64         `json:"memory_usage_bytes"`
	CPUTime         time.Duration `json:"cpu_time"`
	FilesProcessed  int           `json:"files_processed"`
	ViolationsFound int           `json:"violations_found"`
	Success         bool          `json:"success"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

// BenchmarkComparison represents a comparison between two benchmark results
type BenchmarkComparison struct {
	GoMDLint      BenchmarkResult `json:"gomdlint"`
	MarkdownLint  BenchmarkResult `json:"markdownlint"`
	SpeedupFactor float64         `json:"speedup_factor"`
	MemoryRatio   float64         `json:"memory_ratio"`
	Timestamp     time.Time       `json:"timestamp"`
}

// NewBenchmarkCommand creates the benchmark command
func NewBenchmarkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "benchmark [files...]",
		Short: "Performance benchmark against markdownlint",
		Long: `Run performance benchmarks comparing gomdlint against the original Node.js markdownlint.

This command will:
- Test both tools against the same set of markdown files
- Measure execution time, memory usage, and CPU consumption
- Display detailed performance comparison with speedup factors
- Generate test files if none are provided

Examples:
  gomdlint benchmark docs/*.md
  gomdlint benchmark --iterations 5 --generate-test-files
  gomdlint benchmark --output benchmark-results.json
  gomdlint benchmark --markdownlint-cli markdownlint-cli2`,
		Args: cobra.ArbitraryArgs,
		RunE: runBenchmark,
	}

	cmd.Flags().Int("iterations", 3, "Number of benchmark iterations to run")
	cmd.Flags().Bool("generate-test-files", false, "Generate test markdown files for benchmarking")
	cmd.Flags().Int("test-file-count", 50, "Number of test files to generate")
	cmd.Flags().Int("test-file-size", 1000, "Average size of test files in bytes")
	cmd.Flags().String("markdownlint-cli", "markdownlint", "Command to use for markdownlint (markdownlint, markdownlint-cli2)")
	cmd.Flags().Bool("skip-markdownlint", false, "Skip markdownlint benchmark (gomdlint only)")
	cmd.Flags().String("output", "", "Output benchmark results to JSON file")
	cmd.Flags().Bool("verbose", false, "Verbose benchmark output")

	return cmd
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse flags
	iterations, _ := cmd.Flags().GetInt("iterations")
	generateTestFiles, _ := cmd.Flags().GetBool("generate-test-files")
	testFileCount, _ := cmd.Flags().GetInt("test-file-count")
	testFileSize, _ := cmd.Flags().GetInt("test-file-size")
	markdownlintCmd, _ := cmd.Flags().GetString("markdownlint-cli")
	skipMarkdownlint, _ := cmd.Flags().GetBool("skip-markdownlint")
	outputFile, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Setup themed output for benchmarking
	themeService := service.NewThemeService()
	themeConfig := value.NewThemeConfig()
	themedOutput, err := output.NewThemedOutput(ctx, themeConfig, themeService)
	if err != nil {
		// Fall back to default theme on error
		defaultTheme := value.NewThemeConfig()
		themedOutput, _ = output.NewThemedOutput(ctx, defaultTheme, themeService)
	}

	themedOutput.Benchmark("gomdlint Performance Benchmark")
	fmt.Println("================================")

	// Determine files to benchmark
	var files []string

	if generateTestFiles {
		themedOutput.FileFound("Generating %d test files...", testFileCount)
		files, err = generateTestMarkdownFiles(testFileCount, testFileSize)
		if err != nil {
			return fmt.Errorf("failed to generate test files: %w", err)
		}
		defer cleanupTestFiles(files)
	} else if len(args) > 0 {
		files, err = expandFilePaths(args)
		if err != nil {
			return fmt.Errorf("failed to expand file paths: %w", err)
		}
	} else {
		// Look for existing markdown files
		files, err = findMarkdownFiles(".")
		if err != nil || len(files) == 0 {
			fmt.Println("No markdown files found. Generating test files...")
			files, err = generateTestMarkdownFiles(10, testFileSize)
			if err != nil {
				return fmt.Errorf("failed to generate test files: %w", err)
			}
			defer cleanupTestFiles(files)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to benchmark")
	}

	themedOutput.Performance("Benchmarking %d files across %d iterations\n", len(files), iterations)

	// Run gomdlint benchmarks
	themedOutput.Launch("Running gomdlint benchmarks...")
	gomdlintResults := make([]BenchmarkResult, iterations)
	for i := 0; i < iterations; i++ {
		if verbose {
			fmt.Printf("  Iteration %d/%d\n", i+1, iterations)
		}
		result, err := benchmarkGoMDLint(ctx, files)
		if err != nil {
			themedOutput.Warning("  gomdlint iteration %d failed: %v", i+1, err)
			result.Success = false
			result.ErrorMessage = err.Error()
		}
		gomdlintResults[i] = result
	}

	// Run markdownlint benchmarks
	var markdownlintResults []BenchmarkResult
	if !skipMarkdownlint {
		themedOutput.Search("Running markdownlint benchmarks...")

		// Check if markdownlint is available
		if !isCommandAvailable(markdownlintCmd) {
			themedOutput.Warning("%s not found. Install with: npm install -g %s", markdownlintCmd, markdownlintCmd)
			fmt.Println("   Continuing with gomdlint-only benchmark...")
			skipMarkdownlint = true
		} else {
			markdownlintResults = make([]BenchmarkResult, iterations)
			for i := 0; i < iterations; i++ {
				if verbose {
					fmt.Printf("  Iteration %d/%d\n", i+1, iterations)
				}
				result, err := benchmarkMarkdownLint(files, markdownlintCmd)
				if err != nil {
					themedOutput.Warning("  markdownlint iteration %d failed: %v", i+1, err)
					result.Success = false
					result.ErrorMessage = err.Error()
				}
				markdownlintResults[i] = result
			}
		}
	}

	// Calculate averages
	avgGomdlint := calculateAverage(gomdlintResults)

	var comparison *BenchmarkComparison
	if !skipMarkdownlint && len(markdownlintResults) > 0 {
		avgMarkdownlint := calculateAverage(markdownlintResults)
		comparison = &BenchmarkComparison{
			GoMDLint:      avgGomdlint,
			MarkdownLint:  avgMarkdownlint,
			SpeedupFactor: calculateSpeedup(avgMarkdownlint.ExecutionTime, avgGomdlint.ExecutionTime),
			MemoryRatio:   calculateMemoryRatio(avgMarkdownlint.MemoryUsage, avgGomdlint.MemoryUsage),
			Timestamp:     time.Now(),
		}
	}

	// Display results
	displayBenchmarkResults(themedOutput, avgGomdlint, markdownlintResults, comparison, verbose)

	// Save to file if requested
	if outputFile != "" {
		if err := saveBenchmarkResults(outputFile, comparison, avgGomdlint); err != nil {
			return fmt.Errorf("failed to save benchmark results: %w", err)
		}
		themedOutput.FileSaved("Results saved to %s", outputFile)
	}

	return nil
}

// benchmarkGoMDLint runs a benchmark against gomdlint
func benchmarkGoMDLint(ctx context.Context, files []string) (BenchmarkResult, error) {
	result := BenchmarkResult{
		Tool:           "gomdlint",
		Version:        gomdlint.GetVersion(),
		FilesProcessed: len(files),
	}

	// Measure memory before
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Measure execution time
	startTime := time.Now()

	// Run gomdlint
	lintResult, err := gomdlint.LintFiles(ctx, files)
	if err != nil {
		return result, err
	}

	// Measure after
	endTime := time.Now()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	result.ExecutionTime = endTime.Sub(startTime)
	result.CPUTime = result.ExecutionTime // Simplified CPU time measurement
	result.MemoryUsage = int64(memAfter.Alloc - memBefore.Alloc)
	result.ViolationsFound = lintResult.TotalViolations
	result.Success = true

	return result, nil
}

// benchmarkMarkdownLint runs a benchmark against the Node.js markdownlint
func benchmarkMarkdownLint(files []string, command string) (BenchmarkResult, error) {
	result := BenchmarkResult{
		Tool:           "markdownlint",
		FilesProcessed: len(files),
	}

	// Get markdownlint version
	version, err := getMarkdownLintVersion(command)
	if err != nil {
		version = "unknown"
	}
	result.Version = version

	// Prepare command arguments
	args := append([]string{}, files...)

	// Measure execution time and run command
	startTime := time.Now()
	cmd := exec.Command(command, args...)

	// Capture output to count violations
	output, err := cmd.CombinedOutput()
	endTime := time.Now()

	result.ExecutionTime = endTime.Sub(startTime)

	// Get process resource usage if possible
	if cmd.ProcessState != nil {
		if usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
			result.CPUTime = time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
			result.MemoryUsage = usage.Maxrss * 1024 // Convert from KB to bytes on Linux
			if runtime.GOOS == "darwin" {
				result.MemoryUsage = usage.Maxrss // Already in bytes on macOS
			}
		}
	}

	if err != nil {
		// markdownlint exits with non-zero code when violations are found
		if _, ok := err.(*exec.ExitError); ok {
			result.Success = true
			// Count violations from output
			result.ViolationsFound = countViolationsFromOutput(string(output))
		} else {
			return result, err
		}
	} else {
		result.Success = true
		result.ViolationsFound = 0 // No violations found
	}

	return result, nil
}

// Helper functions

func generateTestMarkdownFiles(count, avgSize int) ([]string, error) {
	var files []string
	testDir := "benchmark_test_files"

	if err := os.MkdirAll(testDir, 0755); err != nil {
		return nil, err
	}

	templates := []func(int, int) string{
		generateMarkdownWithViolations,
		generateCompliantMarkdown,
		generateComplexMarkdown,
		generateListHeavyMarkdown,
		generateCodeHeavyMarkdown,
	}

	for i := 0; i < count; i++ {
		filename := filepath.Join(testDir, fmt.Sprintf("test_%03d.md", i))
		template := templates[i%len(templates)]
		content := template(i, avgSize)

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return nil, err
		}
		files = append(files, filename)
	}

	return files, nil
}

func generateMarkdownWithViolations(index, size int) string {
	content := fmt.Sprintf("#Missing space in heading %d\n\n", index)
	content += "This line has	hard tabs and is intentionally very long to trigger the line length violation rule because we want to test performance.\n\n"
	content += "##Another heading without space\n\n"
	content += "- List item\n   - Inconsistent indentation\n- Another item\n\n"

	// Add more content to reach target size
	for len(content) < size {
		content += fmt.Sprintf("Additional paragraph %d with some content to fill space. ", len(content)/50)
		if len(content)%200 == 0 {
			content += "\n\n"
		}
	}

	return content
}

func generateCompliantMarkdown(index, size int) string {
	content := fmt.Sprintf("# Compliant Document %d\n\n", index)
	content += "This is a well-formatted markdown document that should not trigger any violations.\n\n"
	content += "## Section 1\n\nContent here.\n\n"
	content += "### Subsection\n\nMore content.\n\n"
	content += "- Properly formatted list\n- Another item\n- Third item\n\n"

	for len(content) < size {
		content += "This is additional compliant content. "
		if len(content)%100 == 0 {
			content += "\n\n"
		}
	}

	return content
}

func generateComplexMarkdown(index, size int) string {
	content := fmt.Sprintf("# Complex Document %d\n\n", index)
	content += "This document contains various markdown elements:\n\n"
	content += "## Links and Images\n\n"
	content += "[Link text](https://example.com)\n"
	content += "![Alt text](image.png)\n\n"
	content += "## Code Blocks\n\n"
	content += "```go\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\n"
	content += "## Tables\n\n"
	content += "| Column 1 | Column 2 |\n"
	content += "|----------|----------|\n"
	content += "| Data 1   | Data 2   |\n\n"

	for len(content) < size {
		content += fmt.Sprintf("Complex paragraph %d with **bold** and *italic* text. ", len(content)/100)
		if len(content)%150 == 0 {
			content += "\n\n"
		}
	}

	return content
}

func generateListHeavyMarkdown(index, size int) string {
	content := fmt.Sprintf("# List Heavy Document %d\n\n", index)

	for i := 0; len(content) < size; i++ {
		content += fmt.Sprintf("- Item %d\n", i+1)
		if i%5 == 4 {
			content += "\n"
		}
	}

	return content
}

func generateCodeHeavyMarkdown(index, size int) string {
	content := fmt.Sprintf("# Code Heavy Document %d\n\n", index)

	codeBlocks := []string{
		"```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\n",
		"```javascript\nfunction hello() {\n    console.log('Hello, World!');\n}\n```\n\n",
		"```python\ndef hello():\n    print('Hello, World!')\n```\n\n",
		"```bash\necho \"Hello, World!\"\n```\n\n",
	}

	for i := 0; len(content) < size; i++ {
		content += codeBlocks[i%len(codeBlocks)]
	}

	return content
}

func cleanupTestFiles(files []string) {
	if len(files) > 0 {
		testDir := filepath.Dir(files[0])
		os.RemoveAll(testDir)
	}
}

func expandFilePaths(args []string) ([]string, error) {
	var files []string
	for _, pattern := range args {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}

func findMarkdownFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isMarkdownFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func getMarkdownLintVersion(command string) (string, error) {
	cmd := exec.Command(command, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func countViolationsFromOutput(output string) int {
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, ":") && (strings.Contains(line, "MD") || strings.Contains(line, "error")) {
			count++
		}
	}
	return count
}

func calculateAverage(results []BenchmarkResult) BenchmarkResult {
	if len(results) == 0 {
		return BenchmarkResult{}
	}

	avg := BenchmarkResult{
		Tool:    results[0].Tool,
		Version: results[0].Version,
	}

	var totalTime time.Duration
	var totalMemory int64
	var totalCPU time.Duration
	var totalViolations int
	var totalFiles int
	successCount := 0

	for _, result := range results {
		if result.Success {
			totalTime += result.ExecutionTime
			totalMemory += result.MemoryUsage
			totalCPU += result.CPUTime
			totalViolations += result.ViolationsFound
			totalFiles += result.FilesProcessed
			successCount++
		}
	}

	if successCount > 0 {
		avg.ExecutionTime = totalTime / time.Duration(successCount)
		avg.MemoryUsage = totalMemory / int64(successCount)
		avg.CPUTime = totalCPU / time.Duration(successCount)
		avg.ViolationsFound = totalViolations / successCount
		avg.FilesProcessed = totalFiles / successCount
		avg.Success = true
	}

	return avg
}

func calculateSpeedup(oldTime, newTime time.Duration) float64 {
	if newTime == 0 {
		return 0
	}
	return float64(oldTime) / float64(newTime)
}

func calculateMemoryRatio(oldMem, newMem int64) float64 {
	if newMem == 0 {
		return 0
	}
	return float64(oldMem) / float64(newMem)
}

func displayBenchmarkResults(themedOutput *output.ThemedOutput, gomdlint BenchmarkResult, markdownlintResults []BenchmarkResult, comparison *BenchmarkComparison, verbose bool) {
	themedOutput.Results("\nBenchmark Results")
	fmt.Println("===================")

	// Display gomdlint results
	themedOutput.Launch("gomdlint %s:", gomdlint.Version)
	fmt.Printf("  Execution Time: %v\n", gomdlint.ExecutionTime)
	fmt.Printf("  Memory Usage:   %s\n", formatBytes(gomdlint.MemoryUsage))
	fmt.Printf("  Files:          %d\n", gomdlint.FilesProcessed)
	fmt.Printf("  Violations:     %d\n", gomdlint.ViolationsFound)

	if comparison != nil {
		themedOutput.Search("\nmarkdownlint %s:", comparison.MarkdownLint.Version)
		fmt.Printf("  Execution Time: %v\n", comparison.MarkdownLint.ExecutionTime)
		fmt.Printf("  Memory Usage:   %s\n", formatBytes(comparison.MarkdownLint.MemoryUsage))
		fmt.Printf("  Files:          %d\n", comparison.MarkdownLint.FilesProcessed)
		fmt.Printf("  Violations:     %d\n", comparison.MarkdownLint.ViolationsFound)

		themedOutput.Winner("\nPerformance Comparison:")
		fmt.Printf("  Speed Improvement: %.1fx faster\n", comparison.SpeedupFactor)
		if comparison.MemoryRatio > 1 {
			fmt.Printf("  Memory Efficiency: %.1fx less memory\n", comparison.MemoryRatio)
		} else {
			fmt.Printf("  Memory Usage:      %.1fx more memory\n", 1/comparison.MemoryRatio)
		}

		timeSaved := comparison.MarkdownLint.ExecutionTime - comparison.GoMDLint.ExecutionTime
		fmt.Printf("  Time Saved:        %v (%.1f%% reduction)\n",
			timeSaved,
			float64(timeSaved)/float64(comparison.MarkdownLint.ExecutionTime)*100)
	}

	if verbose && len(markdownlintResults) > 0 {
		themedOutput.Performance("\nDetailed Results:")
		fmt.Println("  Iteration | gomdlint | markdownlint | Speedup")
		fmt.Println("  ----------|----------|--------------|--------")
		for i := 0; i < len(markdownlintResults); i++ {
			speedup := calculateSpeedup(markdownlintResults[i].ExecutionTime, gomdlint.ExecutionTime)
			fmt.Printf("  %9d | %8v | %12v | %6.1fx\n",
				i+1,
				gomdlint.ExecutionTime,
				markdownlintResults[i].ExecutionTime,
				speedup)
		}
	}
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

func saveBenchmarkResults(filename string, comparison *BenchmarkComparison, gomdlint BenchmarkResult) error {
	var data interface{}
	if comparison != nil {
		data = comparison
	} else {
		data = map[string]interface{}{
			"gomdlint":  gomdlint,
			"timestamp": time.Now(),
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}
