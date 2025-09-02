package service

import (
	"context"
	"sync"
	"time"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// AsyncRuleEngine extends the base rule engine with async capabilities
type AsyncRuleEngine struct {
	*RuleEngine
	asyncRules     []*entity.AsyncRule
	workerPool     chan struct{} // Semaphore for concurrency control
	resultBuffer   chan entity.AsyncRuleResult
	maxConcurrency int
}

// NewAsyncRuleEngine creates a new async rule engine
func NewAsyncRuleEngine(maxConcurrency int) (*AsyncRuleEngine, error) {
	baseEngine, err := NewRuleEngine()
	if err != nil {
		return nil, err
	}

	return &AsyncRuleEngine{
		RuleEngine:     baseEngine,
		asyncRules:     make([]*entity.AsyncRule, 0),
		workerPool:     make(chan struct{}, maxConcurrency),
		resultBuffer:   make(chan entity.AsyncRuleResult, maxConcurrency*2),
		maxConcurrency: maxConcurrency,
	}, nil
}

// RegisterAsyncRule adds an async rule to the engine
func (are *AsyncRuleEngine) RegisterAsyncRule(rule *entity.AsyncRule) {
	are.asyncRules = append(are.asyncRules, rule)
}

// GetAsyncRules returns all registered async rules
func (are *AsyncRuleEngine) GetAsyncRules() []*entity.AsyncRule {
	return are.asyncRules
}

// LintDocumentAsync executes all rules concurrently and returns a channel of results
func (are *AsyncRuleEngine) LintDocumentAsync(
	ctx context.Context,
	tokens []value.Token,
	lines []string,
	filename string,
) <-chan entity.AsyncRuleResult {
	resultChan := make(chan entity.AsyncRuleResult, are.maxConcurrency)

	go func() {
		defer close(resultChan)

		var wg sync.WaitGroup

		// Execute sync rules in parallel
		syncRules := are.GetEnabledRules()
		for _, rule := range syncRules {
			if !are.IsRuleEnabled(rule.PrimaryName()) {
				continue
			}

			wg.Add(1)
			go func(r *entity.Rule) {
				defer wg.Done()

				// Acquire worker slot
				select {
				case are.workerPool <- struct{}{}:
					defer func() { <-are.workerPool }()
				case <-ctx.Done():
					return
				}

				params := entity.RuleParams{
					Lines:    lines,
					Config:   are.GetRuleConfig(r.PrimaryName()),
					Filename: filename,
					Tokens:   tokens,
				}

				start := time.Now()
				result := r.Execute(ctx, params)
				duration := time.Since(start)

				asyncResult := entity.AsyncRuleResult{
					Metadata: map[string]interface{}{
						"rule":      r.PrimaryName(),
						"sync":      true,
						"timestamp": time.Now(),
						"duration":  duration,
						"filename":  filename,
					},
				}

				if result.IsErr() {
					asyncResult.Error = result.Error()
				} else {
					asyncResult.Violations = result.Unwrap()
				}

				select {
				case resultChan <- asyncResult:
				case <-ctx.Done():
					return
				}
			}(rule)
		}

		// Execute async rules
		for _, asyncRule := range are.asyncRules {
			if !are.IsRuleEnabled(asyncRule.PrimaryName()) {
				continue
			}

			wg.Add(1)
			go func(ar *entity.AsyncRule) {
				defer wg.Done()

				params := entity.RuleParams{
					Lines:    lines,
					Config:   are.GetRuleConfig(ar.PrimaryName()),
					Filename: filename,
					Tokens:   tokens,
				}

				start := time.Now()
				asyncResultChan := ar.ExecuteAsync(ctx, params)

				select {
				case result := <-asyncResultChan:
					duration := time.Since(start)

					if result.Metadata == nil {
						result.Metadata = make(map[string]interface{})
					}

					result.Metadata["rule"] = ar.PrimaryName()
					result.Metadata["sync"] = false
					result.Metadata["timestamp"] = time.Now()
					result.Metadata["duration"] = duration
					result.Metadata["filename"] = filename

					select {
					case resultChan <- result:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					// Handle cancellation
					asyncResult := entity.AsyncRuleResult{
						Error: ctx.Err(),
						Metadata: map[string]interface{}{
							"rule":      ar.PrimaryName(),
							"sync":      false,
							"timestamp": time.Now(),
							"duration":  time.Since(start),
							"filename":  filename,
							"cancelled": true,
						},
					}

					select {
					case resultChan <- asyncResult:
					case <-time.After(100 * time.Millisecond):
						// Avoid blocking on channel send during shutdown
					}
					return
				}
			}(asyncRule)
		}

		wg.Wait()
	}()

	return resultChan
}

// LintDocumentAsyncBlocking executes all rules and collects results
func (are *AsyncRuleEngine) LintDocumentAsyncBlocking(
	ctx context.Context,
	tokens []value.Token,
	lines []string,
	filename string,
) functional.Result[*AsyncLintResult] {
	resultChan := are.LintDocumentAsync(ctx, tokens, lines, filename)

	var allViolations []value.Violation
	var errors []error
	ruleResults := make(map[string]entity.AsyncRuleResult)

	for result := range resultChan {
		if result.Error != nil {
			errors = append(errors, result.Error)
		} else {
			allViolations = append(allViolations, result.Violations...)
		}

		if ruleName, ok := result.Metadata["rule"].(string); ok {
			ruleResults[ruleName] = result
		}
	}

	asyncResult := &AsyncLintResult{
		Violations:  allViolations,
		Errors:      errors,
		RuleResults: ruleResults,
		Metadata: map[string]interface{}{
			"filename":      filename,
			"total_rules":   len(ruleResults),
			"total_errors":  len(errors),
			"timestamp":     time.Now(),
			"sync_rules":    len(are.GetEnabledRules()),
			"async_rules":   len(are.asyncRules),
			"concurrency":   are.maxConcurrency,
		},
	}

	if len(errors) > 0 {
		return functional.Err[*AsyncLintResult](errors[0]) // Return first error
	}

	return functional.Ok(asyncResult)
}

// AsyncLintResult contains the complete results of async linting
type AsyncLintResult struct {
	Violations  []value.Violation
	Errors      []error
	RuleResults map[string]entity.AsyncRuleResult
	Metadata    map[string]interface{}
}

// GetViolationCount returns the total number of violations
func (alr *AsyncLintResult) GetViolationCount() int {
	return len(alr.Violations)
}

// GetErrorCount returns the total number of errors
func (alr *AsyncLintResult) GetErrorCount() int {
	return len(alr.Errors)
}

// GetRuleResult returns the result for a specific rule
func (alr *AsyncLintResult) GetRuleResult(ruleName string) (entity.AsyncRuleResult, bool) {
	result, exists := alr.RuleResults[ruleName]
	return result, exists
}

// GetSuccessfulRules returns rules that executed without errors
func (alr *AsyncLintResult) GetSuccessfulRules() []string {
	var successful []string
	for ruleName, result := range alr.RuleResults {
		if result.Error == nil {
			successful = append(successful, ruleName)
		}
	}
	return successful
}

// GetFailedRules returns rules that encountered errors
func (alr *AsyncLintResult) GetFailedRules() []string {
	var failed []string
	for ruleName, result := range alr.RuleResults {
		if result.Error != nil {
			failed = append(failed, ruleName)
		}
	}
	return failed
}

// GetTotalDuration calculates total execution time across all rules
func (alr *AsyncLintResult) GetTotalDuration() time.Duration {
	var total time.Duration
	for _, result := range alr.RuleResults {
		if duration, ok := result.Metadata["duration"].(time.Duration); ok {
			total += duration
		}
	}
	return total
}

// SetMaxConcurrency updates the maximum concurrency level
func (are *AsyncRuleEngine) SetMaxConcurrency(max int) {
	are.maxConcurrency = max
	// Create new worker pool with updated size
	are.workerPool = make(chan struct{}, max)
	are.resultBuffer = make(chan entity.AsyncRuleResult, max*2)
}

// GetMaxConcurrency returns the current maximum concurrency level
func (are *AsyncRuleEngine) GetMaxConcurrency() int {
	return are.maxConcurrency
}

// GetStats returns async engine statistics
func (are *AsyncRuleEngine) GetAsyncStats() map[string]interface{} {
	baseStats := are.RuleEngine.Stats()
	
	baseStats["async_rules"] = len(are.asyncRules)
	baseStats["max_concurrency"] = are.maxConcurrency
	baseStats["worker_pool_size"] = cap(are.workerPool)
	baseStats["result_buffer_size"] = cap(are.resultBuffer)

	return baseStats
}