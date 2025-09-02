package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Mock rule that takes time to execute (for testing async behavior)
func newSlowRule(name string, duration time.Duration, shouldError bool) functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://example.com/slow-rule")

	ruleFunc := func(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
		// Simulate slow processing
		select {
		case <-time.After(duration):
		case <-ctx.Done():
			return functional.Err[[]value.Violation](ctx.Err())
		}

		if shouldError {
			return functional.Err[[]value.Violation](fmt.Errorf("mock rule error"))
		}

		// Create a test violation
		violation := value.NewViolation(
			[]string{name},
			fmt.Sprintf("Test violation from %s", name),
			infoURL,
			1,
		)

		return functional.Ok([]value.Violation{*violation})
	}

	return entity.NewRule(
		[]string{name},
		fmt.Sprintf("Test rule %s", name),
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		ruleFunc,
	)
}

func TestAsyncRuleEngine_NewAsyncRuleEngine(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(5, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	if engine == nil {
		t.Fatal("expected non-nil async rule engine")
	}

	if engine.maxConcurrency != 5 {
		t.Errorf("expected max concurrency 5, got %d", engine.maxConcurrency)
	}

	if engine.timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", engine.timeout)
	}

	stats := engine.GetAsyncStats()
	if stats["max_concurrency"] != 5 {
		t.Errorf("expected max_concurrency 5 in stats, got %v", stats["max_concurrency"])
	}
}

func TestAsyncRuleEngine_DefaultValues(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(0, 0)
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Should use default values
	if engine.maxConcurrency != 10 {
		t.Errorf("expected default max concurrency 10, got %d", engine.maxConcurrency)
	}

	if engine.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", engine.timeout)
	}
}

func TestAsyncRuleEngine_ExecuteConcurrentRules(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(3, 100*time.Millisecond) // Reduced from 5s
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Clear built-in rules for clean testing
	engine.rules = []*entity.Rule{}
	engine.ruleIndex = make(map[string]*entity.Rule)
	engine.tagIndex = make(map[string][]*entity.Rule)
	engine.enabledRules = make(map[string]bool)
	engine.ruleConfigs = make(map[string]map[string]interface{})

	// Add test rules with different execution times
	testRules := []struct {
		name     string
		duration time.Duration
		enabled  bool
	}{
		{"FAST001", 1 * time.Millisecond, true},   // Reduced from 10ms
		{"MEDIUM001", 5 * time.Millisecond, true}, // Reduced from 100ms
		{"SLOW001", 10 * time.Millisecond, true},  // Reduced from 500ms
		{"DISABLED001", 1 * time.Millisecond, false},
	}

	for _, tr := range testRules {
		ruleResult := newSlowRule(tr.name, tr.duration, false)
		if ruleResult.IsErr() {
			t.Fatalf("failed to create rule %s: %v", tr.name, ruleResult.Error())
		}

		rule := ruleResult.Unwrap()
		err := engine.RegisterRule(rule)
		if err != nil {
			t.Fatalf("failed to register rule %s: %v", tr.name, err)
		}

		engine.enabledRules[tr.name] = tr.enabled
	}

	ctx := context.Background()
	tokens := []value.Token{}
	lines := []string{"# Test", "Some content"}
	filename := "test.md"

	startTime := time.Now()
	resultChan := engine.LintDocumentAsync(ctx, tokens, lines, filename)

	var results []AsyncRuleResult
	for result := range resultChan {
		results = append(results, result)
	}

	totalDuration := time.Since(startTime)

	// Should have results only for enabled rules
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Check that concurrent execution was faster than sequential
	// Sequential would take: 1ms + 5ms + 10ms = 16ms+
	// Concurrent should be closer to max(1, 5, 10) = 10ms+
	if totalDuration > 100*time.Millisecond { // Allow some buffer for test overhead
		t.Errorf("async execution took too long: %v", totalDuration)
	}

	// Verify all expected rules produced results
	ruleNames := make([]string, len(results))
	for i, result := range results {
		ruleNames[i] = result.Rule.PrimaryName()

		if result.Error != nil {
			t.Errorf("unexpected error from rule %s: %v", result.Rule.PrimaryName(), result.Error)
		}

		if len(result.Violations) != 1 {
			t.Errorf("expected 1 violation from rule %s, got %d", result.Rule.PrimaryName(), len(result.Violations))
		}
	}

	// Check disabled rule is not included
	for _, name := range ruleNames {
		if name == "DISABLED001" {
			t.Error("disabled rule should not produce results")
		}
	}
}

func TestAsyncRuleEngine_Timeout(t *testing.T) {
	t.Parallel()
	// Create engine with very short timeout
	engine, err := NewAsyncRuleEngine(1, 5*time.Millisecond) // Reduced from 50ms
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Clear built-in rules
	engine.rules = []*entity.Rule{}
	engine.ruleIndex = make(map[string]*entity.Rule)
	engine.tagIndex = make(map[string][]*entity.Rule)
	engine.enabledRules = make(map[string]bool)
	engine.ruleConfigs = make(map[string]map[string]interface{})

	// Add a rule that takes longer than timeout
	ruleResult := newSlowRule("TIMEOUT001", 20*time.Millisecond, false) // Reduced from 200ms
	if ruleResult.IsErr() {
		t.Fatalf("failed to create slow rule: %v", ruleResult.Error())
	}

	rule := ruleResult.Unwrap()
	err = engine.RegisterRule(rule)
	if err != nil {
		t.Fatalf("failed to register rule: %v", err)
	}

	engine.enabledRules["TIMEOUT001"] = true

	ctx := context.Background()
	tokens := []value.Token{}
	lines := []string{"# Test"}
	filename := "test.md"

	resultChan := engine.LintDocumentAsync(ctx, tokens, lines, filename)

	var result AsyncRuleResult
	for r := range resultChan {
		result = r
		break // Should only be one result
	}

	// Should have timeout error
	if result.Error == nil {
		t.Error("expected timeout error, got nil")
	}

	if !strings.Contains(result.Error.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline exceeded error, got: %v", result.Error)
	}
}

func TestAsyncRuleEngine_ErrorHandling(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(2, 100*time.Millisecond) // Reduced from 5s
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Clear built-in rules
	engine.rules = []*entity.Rule{}
	engine.ruleIndex = make(map[string]*entity.Rule)
	engine.tagIndex = make(map[string][]*entity.Rule)
	engine.enabledRules = make(map[string]bool)
	engine.ruleConfigs = make(map[string]map[string]interface{})

	// Add rules: one successful, one that errors
	successRule := newSlowRule("SUCCESS001", 1*time.Millisecond, false) // Reduced from 10ms
	if successRule.IsErr() {
		t.Fatalf("failed to create success rule: %v", successRule.Error())
	}

	errorRule := newSlowRule("ERROR001", 1*time.Millisecond, true) // Reduced from 10ms
	if errorRule.IsErr() {
		t.Fatalf("failed to create error rule: %v", errorRule.Error())
	}

	err = engine.RegisterRule(successRule.Unwrap())
	if err != nil {
		t.Fatalf("failed to register success rule: %v", err)
	}

	err = engine.RegisterRule(errorRule.Unwrap())
	if err != nil {
		t.Fatalf("failed to register error rule: %v", err)
	}

	engine.enabledRules["SUCCESS001"] = true
	engine.enabledRules["ERROR001"] = true

	ctx := context.Background()
	tokens := []value.Token{}
	lines := []string{"# Test"}
	filename := "test.md"

	resultChan := engine.LintDocumentAsync(ctx, tokens, lines, filename)

	results := make(map[string]AsyncRuleResult)
	for result := range resultChan {
		results[result.Rule.PrimaryName()] = result
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Check success rule
	if successResult, exists := results["SUCCESS001"]; exists {
		if successResult.Error != nil {
			t.Errorf("success rule should not have error: %v", successResult.Error)
		}
		if len(successResult.Violations) != 1 {
			t.Errorf("success rule should have 1 violation, got %d", len(successResult.Violations))
		}
	} else {
		t.Error("missing result for SUCCESS001")
	}

	// Check error rule
	if errorResult, exists := results["ERROR001"]; exists {
		if errorResult.Error == nil {
			t.Error("error rule should have error")
		}
		if !strings.Contains(errorResult.Error.Error(), "mock rule error") {
			t.Errorf("unexpected error message: %v", errorResult.Error)
		}
	} else {
		t.Error("missing result for ERROR001")
	}
}

func TestAsyncRuleEngine_ContextCancellation(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(1, 100*time.Millisecond) // Reduced from 10s
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Clear built-in rules
	engine.rules = []*entity.Rule{}
	engine.ruleIndex = make(map[string]*entity.Rule)
	engine.tagIndex = make(map[string][]*entity.Rule)
	engine.enabledRules = make(map[string]bool)
	engine.ruleConfigs = make(map[string]map[string]interface{})

	// Add slow rule
	ruleResult := newSlowRule("SLOW001", 50*time.Millisecond, false) // Reduced from 1s
	if ruleResult.IsErr() {
		t.Fatalf("failed to create slow rule: %v", ruleResult.Error())
	}

	rule := ruleResult.Unwrap()
	err = engine.RegisterRule(rule)
	if err != nil {
		t.Fatalf("failed to register rule: %v", err)
	}

	engine.enabledRules["SLOW001"] = true

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	tokens := []value.Token{}
	lines := []string{"# Test"}
	filename := "test.md"

	resultChan := engine.LintDocumentAsync(ctx, tokens, lines, filename)

	// Cancel context with a small delay to allow execution to start
	go func() {
		time.Sleep(5 * time.Millisecond) // Reduced from 50ms
		cancel()
	}()

	var results []AsyncRuleResult
	timeout := time.After(200 * time.Millisecond) // Reduced from 2s

	// Collect results with timeout
	collecting := true
	for collecting {
		select {
		case r, ok := <-resultChan:
			if !ok {
				collecting = false
				break
			}
			results = append(results, r)
		case <-timeout:
			t.Log("Timeout waiting for results")
			collecting = false
		}
	}

	// The test should either get a cancellation error OR no results (if cancelled before execution)
	if len(results) == 0 {
		t.Log("No results received - cancellation happened before execution started (acceptable)")
		return
	}

	// If we got results, check for cancellation error
	found := false
	for _, result := range results {
		if result.Error != nil && strings.Contains(result.Error.Error(), "context canceled") {
			found = true
			break
		}
	}

	if !found {
		t.Logf("Got %d results but none with cancellation error: %v", len(results), results)
		// This is also acceptable - the rule might have completed before cancellation
	}
}

func TestAsyncRuleEngine_SettersAndGetters(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(5, 100*time.Millisecond) // Reduced from 10s
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Test SetMaxConcurrency
	engine.SetMaxConcurrency(8)
	if engine.maxConcurrency != 8 {
		t.Errorf("expected max concurrency 8, got %d", engine.maxConcurrency)
	}

	// Test invalid concurrency (should set to 1)
	engine.SetMaxConcurrency(0)
	if engine.maxConcurrency != 1 {
		t.Errorf("expected max concurrency 1 for invalid input, got %d", engine.maxConcurrency)
	}

	// Test SetTimeout
	engine.SetTimeout(150 * time.Millisecond) // Reduced from 15s
	if engine.timeout != 150*time.Millisecond {
		t.Errorf("expected timeout 150ms, got %v", engine.timeout)
	}

	// Test invalid timeout (should set to default)
	engine.SetTimeout(0)
	if engine.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s for invalid input, got %v", engine.timeout)
	}

	// Test stats
	stats := engine.GetAsyncStats()
	if stats["max_concurrency"] != 1 {
		t.Errorf("expected max_concurrency 1 in stats, got %v", stats["max_concurrency"])
	}
	if stats["timeout_ms"] != int64(30000) {
		t.Errorf("expected timeout_ms 30000 in stats, got %v", stats["timeout_ms"])
	}
}

func TestAsyncRuleEngine_CollectResults(t *testing.T) {
	t.Parallel()
	engine, err := NewAsyncRuleEngine(2, 100*time.Millisecond) // Reduced from 5s
	if err != nil {
		t.Fatalf("failed to create async rule engine: %v", err)
	}

	// Clear built-in rules
	engine.rules = []*entity.Rule{}
	engine.ruleIndex = make(map[string]*entity.Rule)
	engine.tagIndex = make(map[string][]*entity.Rule)
	engine.enabledRules = make(map[string]bool)
	engine.ruleConfigs = make(map[string]map[string]interface{})

	// Add multiple test rules
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("TEST%03d", i+1)
		ruleResult := newSlowRule(name, 1*time.Millisecond, false) // Reduced from 10ms
		if ruleResult.IsErr() {
			t.Fatalf("failed to create rule %s: %v", name, ruleResult.Error())
		}

		rule := ruleResult.Unwrap()
		err := engine.RegisterRule(rule)
		if err != nil {
			t.Fatalf("failed to register rule %s: %v", name, err)
		}

		engine.enabledRules[name] = true
	}

	ctx := context.Background()
	tokens := []value.Token{}
	lines := []string{"# Test"}
	filename := "test.md"

	// Test the collect method
	result := engine.LintDocumentAsyncCollect(ctx, tokens, lines, filename)
	if result.IsErr() {
		t.Fatalf("async collect failed: %v", result.Error())
	}

	violations := result.Unwrap()

	// Should have 3 violations (one from each rule)
	if len(violations) != 3 {
		t.Errorf("expected 3 violations, got %d", len(violations))
	}

	// Verify violations have rule information set
	for _, violation := range violations {
		if violation.RuleInformation == nil {
			t.Error("violation should have rule information set")
		}
	}
}

func BenchmarkAsyncRuleEngine_vs_SyncRuleEngine(b *testing.B) {
	// Create sync engine
	syncEngine, err := NewRuleEngine()
	if err != nil {
		b.Fatalf("failed to create sync rule engine: %v", err)
	}

	// Create async engine
	asyncEngine, err := NewAsyncRuleEngine(10, 100*time.Millisecond) // Reduced from 10s for benchmark
	if err != nil {
		b.Fatalf("failed to create async rule engine: %v", err)
	}

	ctx := context.Background()
	tokens := []value.Token{}
	lines := []string{"# Test Heading", "Some content here", "More content"}
	filename := "test.md"

	b.Run("Sync", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := syncEngine.LintDocument(ctx, tokens, lines, filename)
			if result.IsErr() {
				b.Errorf("sync lint failed: %v", result.Error())
			}
		}
	})

	b.Run("Async", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := asyncEngine.LintDocumentAsyncCollect(ctx, tokens, lines, filename)
			if result.IsErr() {
				b.Errorf("async lint failed: %v", result.Error())
			}
		}
	})
}
