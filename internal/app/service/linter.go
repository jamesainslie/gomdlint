package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// LinterService provides the main linting functionality.
// It orchestrates parsing, rule execution, and result aggregation.
type LinterService struct {
	parser     *ParserService
	ruleEngine *RuleEngine

	// Configuration
	options *value.LintOptions

	// Performance optimizations
	concurrency int
	resultCache map[string]*value.LintResult
	cacheMutex  sync.RWMutex
}

// NewLinterService creates a new linting service with the specified options.
func NewLinterService(options *value.LintOptions) (*LinterService, error) {
	if options == nil {
		return nil, fmt.Errorf("linting options cannot be nil")
	}

	parser := NewParserService()

	ruleEngine, err := NewRuleEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to create rule engine: %w", err)
	}

	// Configure rules based on options
	if len(options.Config) > 0 {
		if err := ruleEngine.ConfigureRules(options.Config); err != nil {
			return nil, fmt.Errorf("failed to configure rules: %w", err)
		}
	}

	linter := &LinterService{
		parser:      parser,
		ruleEngine:  ruleEngine,
		options:     options,
		concurrency: 4, // Default concurrency
		resultCache: make(map[string]*value.LintResult),
	}

	return linter, nil
}

// LintFiles lints the specified markdown files.
func (ls *LinterService) LintFiles(ctx context.Context, files []string) functional.Result[*value.LintResult] {
	result := value.NewLintResult()

	// Use concurrent processing for better performance
	type fileResult struct {
		identifier string
		violations []value.Violation
		err        error
	}

	fileChan := make(chan string, len(files))
	resultChan := make(chan fileResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	workerCount := ls.concurrency
	if len(files) < workerCount {
		workerCount = len(files)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filename := range fileChan {
				violations, err := ls.lintFile(ctx, filename)
				resultChan <- fileResult{
					identifier: filename,
					violations: violations,
					err:        err,
				}
			}
		}()
	}

	// Send files to workers
	go func() {
		defer close(fileChan)
		for _, file := range files {
			select {
			case fileChan <- file:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	for fileResult := range resultChan {
		if fileResult.err != nil {
			// Create an error violation for files that couldn't be processed
			errorViolation := value.NewViolation(
				[]string{"FILE_ERROR"},
				"File processing error",
				nil,
				1,
			)
			errorViolation = errorViolation.WithErrorDetail(fileResult.err.Error())
			result.AddViolations(fileResult.identifier, []value.Violation{*errorViolation})
		} else {
			result.AddViolations(fileResult.identifier, fileResult.violations)
		}
	}

	return functional.Ok(result)
}

// LintStrings lints the specified string content.
func (ls *LinterService) LintStrings(ctx context.Context, content map[string]string) functional.Result[*value.LintResult] {
	result := value.NewLintResult()

	// Process each string
	for identifier, text := range content {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return functional.Err[*value.LintResult](ctx.Err())
		default:
		}

		violations, err := ls.lintString(ctx, text, identifier)
		if err != nil {
			// Create an error violation for strings that couldn't be processed
			errorViolation := value.NewViolation(
				[]string{"STRING_ERROR"},
				"String processing error",
				nil,
				1,
			)
			errorViolation = errorViolation.WithErrorDetail(err.Error())
			result.AddViolations(identifier, []value.Violation{*errorViolation})
		} else {
			result.AddViolations(identifier, violations)
		}
	}

	return functional.Ok(result)
}

// Lint processes both files and strings according to the configured options.
func (ls *LinterService) Lint(ctx context.Context) functional.Result[*value.LintResult] {
	finalResult := value.NewLintResult()

	// Lint files if specified
	if len(ls.options.Files) > 0 {
		fileResult := ls.LintFiles(ctx, ls.options.Files)
		if fileResult.IsErr() {
			return fileResult
		}

		// Merge file results
		fileResults := fileResult.Unwrap()
		for identifier, violations := range fileResults.Results {
			finalResult.AddViolations(identifier, violations)
		}
	}

	// Lint strings if specified
	if len(ls.options.Strings) > 0 {
		stringResult := ls.LintStrings(ctx, ls.options.Strings)
		if stringResult.IsErr() {
			return stringResult
		}

		// Merge string results
		stringResults := stringResult.Unwrap()
		for identifier, violations := range stringResults.Results {
			finalResult.AddViolations(identifier, violations)
		}
	}

	return functional.Ok(finalResult)
}

// lintFile processes a single file and returns violations.
func (ls *LinterService) lintFile(ctx context.Context, filename string) ([]value.Violation, error) {
	// Check cache first
	ls.cacheMutex.RLock()
	if cached, exists := ls.resultCache[filename]; exists {
		ls.cacheMutex.RUnlock()
		return cached.GetViolations(filename), nil
	}
	ls.cacheMutex.RUnlock()

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return ls.lintString(ctx, string(content), filename)
}

// lintString processes string content and returns violations.
func (ls *LinterService) lintString(ctx context.Context, content string, identifier string) ([]value.Violation, error) {
	// Remove front matter if configured
	processedContent := ls.removeFrontMatter(content)

	// Parse the content
	tokensResult := ls.parser.ParseDocument(ctx, processedContent, identifier)
	if tokensResult.IsErr() {
		return nil, fmt.Errorf("failed to parse content: %w", tokensResult.Error())
	}

	tokens := tokensResult.Unwrap()
	lines := strings.Split(processedContent, "\n")

	// Apply inline configuration comments if enabled
	if !ls.options.NoInlineConfig {
		tokens, lines = ls.processInlineConfig(tokens, lines)
	}

	// Run rules against the parsed content
	violationsResult := ls.ruleEngine.LintDocument(ctx, tokens, lines, identifier)
	if violationsResult.IsErr() {
		return nil, fmt.Errorf("failed to execute rules: %w", violationsResult.Error())
	}

	violations := violationsResult.Unwrap()

	// Filter violations based on inline config (markdownlint-disable comments)
	filteredViolations := ls.filterViolationsByInlineConfig(violations, lines)

	// Cache the result
	ls.cacheMutex.Lock()
	result := value.NewLintResult()
	result.AddViolations(identifier, filteredViolations)
	ls.resultCache[identifier] = result
	ls.cacheMutex.Unlock()

	return filteredViolations, nil
}

// removeFrontMatter removes front matter from the beginning of content.
func (ls *LinterService) removeFrontMatter(content string) string {
	if ls.options.FrontMatter.IsNone() {
		return content
	}

	regex := ls.options.FrontMatter.Unwrap()
	return regex.ReplaceAllString(content, "")
}

// processInlineConfig processes inline configuration comments.
// This is a simplified version - a full implementation would parse
// markdownlint-disable, markdownlint-enable, etc. comments.
func (ls *LinterService) processInlineConfig(tokens []value.Token, lines []string) ([]value.Token, []string) {
	// TODO: Implement full inline config processing
	// This would involve parsing HTML comments like:
	// <!-- markdownlint-disable MD001 -->
	// <!-- markdownlint-enable MD001 -->
	// <!-- markdownlint-disable-next-line MD001 -->
	// etc.

	return tokens, lines
}

// filterViolationsByInlineConfig filters violations based on inline disable/enable comments.
func (ls *LinterService) filterViolationsByInlineConfig(violations []value.Violation, lines []string) []value.Violation {
	// TODO: Implement filtering based on inline comments
	// For now, return all violations
	return violations
}

// GetParserService returns the underlying parser service.
func (ls *LinterService) GetParserService() *ParserService {
	return ls.parser
}

// GetRuleEngine returns the underlying rule engine.
func (ls *LinterService) GetRuleEngine() *RuleEngine {
	return ls.ruleEngine
}

// GetOptions returns the linting options.
func (ls *LinterService) GetOptions() *value.LintOptions {
	return ls.options
}

// UpdateOptions updates the linting options and reconfigures the rule engine.
func (ls *LinterService) UpdateOptions(options *value.LintOptions) error {
	ls.options = options

	// Reconfigure rules if config changed
	if len(options.Config) > 0 {
		if err := ls.ruleEngine.ConfigureRules(options.Config); err != nil {
			return fmt.Errorf("failed to reconfigure rules: %w", err)
		}
	}

	// Clear cache since configuration changed
	ls.cacheMutex.Lock()
	ls.resultCache = make(map[string]*value.LintResult)
	ls.cacheMutex.Unlock()

	return nil
}

// ClearCache clears the internal result cache.
func (ls *LinterService) ClearCache() {
	ls.cacheMutex.Lock()
	ls.resultCache = make(map[string]*value.LintResult)
	ls.cacheMutex.Unlock()

	// Also clear parser caches
	ls.parser.ClearCaches()
}

// Stats returns linting statistics.
func (ls *LinterService) Stats() map[string]interface{} {
	ls.cacheMutex.RLock()
	cacheSize := len(ls.resultCache)
	ls.cacheMutex.RUnlock()

	stats := ls.ruleEngine.Stats()
	stats["cache_size"] = cacheSize
	stats["concurrency"] = ls.concurrency

	return stats
}
