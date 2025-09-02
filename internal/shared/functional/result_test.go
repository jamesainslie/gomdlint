package functional

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOk(t *testing.T) {
	result := Ok(42)
	assert.True(t, result.IsOk())
	assert.False(t, result.IsErr())
	assert.Equal(t, 42, result.Unwrap())
}

func TestErr(t *testing.T) {
	err := errors.New("test error")
	result := Err[int](err)
	assert.False(t, result.IsOk())
	assert.True(t, result.IsErr())
	assert.Equal(t, err, result.Error())
}

func TestResult_Unwrap_Ok(t *testing.T) {
	result := Ok("hello")
	value := result.Unwrap()
	assert.Equal(t, "hello", value)
}

func TestResult_Unwrap_Err_Panics(t *testing.T) {
	result := Err[string](errors.New("test error"))
	assert.Panics(t, func() {
		result.Unwrap()
	})
}

func TestResult_UnwrapOr(t *testing.T) {
	// Test with Ok value
	okResult := Ok(42)
	assert.Equal(t, 42, okResult.UnwrapOr(100))

	// Test with Err value
	errResult := Err[int](errors.New("test error"))
	assert.Equal(t, 100, errResult.UnwrapOr(100))
}

func TestResult_UnwrapOrElse(t *testing.T) {
	fallback := func(err error) int {
		return len(err.Error())
	}

	// Test with Ok value
	okResult := Ok(42)
	assert.Equal(t, 42, okResult.UnwrapOrElse(fallback))

	// Test with Err value
	errResult := Err[int](errors.New("test"))
	assert.Equal(t, 4, errResult.UnwrapOrElse(fallback)) // len("test") = 4
}

func TestResult_Error_Ok_Panics(t *testing.T) {
	result := Ok("hello")
	assert.Panics(t, func() {
		result.Error()
	})
}

func TestResult_Error_Err(t *testing.T) {
	err := errors.New("test error")
	result := Err[string](err)
	assert.Equal(t, err, result.Error())
}

func TestResult_Map(t *testing.T) {
	// Test mapping Ok value
	okResult := Ok(5)
	mapped := MapResult(okResult, func(x int) string {
		return fmt.Sprintf("value: %d", x)
	})
	assert.True(t, mapped.IsOk())
	assert.Equal(t, "value: 5", mapped.Unwrap())

	// Test mapping Err value (should pass through)
	errResult := Err[int](errors.New("test error"))
	mapped = MapResult(errResult, func(x int) string {
		return fmt.Sprintf("value: %d", x)
	})
	assert.True(t, mapped.IsErr())
	assert.Equal(t, "test error", mapped.Error().Error())
}

func TestResult_FlatMap(t *testing.T) {
	// Test flat mapping Ok value that returns Ok
	okResult := Ok(5)
	flatMapped := AndThen(okResult, func(x int) Result[string] {
		return Ok(fmt.Sprintf("value: %d", x))
	})
	assert.True(t, flatMapped.IsOk())
	assert.Equal(t, "value: 5", flatMapped.Unwrap())

	// Test flat mapping Ok value that returns Err
	flatMapped = AndThen(okResult, func(x int) Result[string] {
		return Err[string](errors.New("inner error"))
	})
	assert.True(t, flatMapped.IsErr())
	assert.Equal(t, "inner error", flatMapped.Error().Error())

	// Test flat mapping Err value (should pass through)
	errResult := Err[int](errors.New("outer error"))
	flatMapped = AndThen(errResult, func(x int) Result[string] {
		return Ok("should not be called")
	})
	assert.True(t, flatMapped.IsErr())
	assert.Equal(t, "outer error", flatMapped.Error().Error())
}

func TestResult_MapErr(t *testing.T) {
	// Test mapping Ok value (should pass through)
	okResult := Ok(42)
	mapped := okResult.MapErr(func(err error) error {
		return fmt.Errorf("wrapped: %w", err)
	})
	assert.True(t, mapped.IsOk())
	assert.Equal(t, 42, mapped.Unwrap())

	// Test mapping Err value
	errResult := Err[int](errors.New("original error"))
	mapped = errResult.MapErr(func(err error) error {
		return fmt.Errorf("wrapped: %w", err)
	})
	assert.True(t, mapped.IsErr())
	assert.Contains(t, mapped.Error().Error(), "wrapped: original error")
}

func TestResult_AndThen(t *testing.T) {
	// Test chaining Ok values
	result1 := Ok(5)
	result2 := AndThen(result1, func(x int) Result[int] {
		return Ok(x * 2)
	})
	assert.True(t, result2.IsOk())
	assert.Equal(t, 10, result2.Unwrap())

	// Test chaining where first is Ok, second is Err
	result2 = AndThen(result1, func(x int) Result[int] {
		return Err[int](errors.New("chain error"))
	})
	assert.True(t, result2.IsErr())
	assert.Equal(t, "chain error", result2.Error().Error())

	// Test chaining where first is Err
	errResult := Err[int](errors.New("initial error"))
	result2 = AndThen(errResult, func(x int) Result[int] {
		return Ok(x * 2)
	})
	assert.True(t, result2.IsErr())
	assert.Equal(t, "initial error", result2.Error().Error())
}

func TestResult_OrElse(t *testing.T) {
	// Test with Ok value (should return original)
	okResult := Ok(42)
	result := okResult.OrElse(func(err error) Result[int] {
		return Ok(100)
	})
	assert.True(t, result.IsOk())
	assert.Equal(t, 42, result.Unwrap())

	// Test with Err value that recovers
	errResult := Err[int](errors.New("original error"))
	result = errResult.OrElse(func(err error) Result[int] {
		return Ok(100)
	})
	assert.True(t, result.IsOk())
	assert.Equal(t, 100, result.Unwrap())

	// Test with Err value that returns another Err
	result = errResult.OrElse(func(err error) Result[int] {
		return Err[int](errors.New("recovery failed"))
	})
	assert.True(t, result.IsErr())
	assert.Equal(t, "recovery failed", result.Error().Error())
}

func TestResult_String(t *testing.T) {
	// Test Ok result string representation
	okResult := Ok(42)
	str := okResult.String()
	assert.Contains(t, str, "Ok")
	assert.Contains(t, str, "42")

	// Test Err result string representation
	errResult := Err[int](errors.New("test error"))
	str = errResult.String()
	assert.Contains(t, str, "Err")
	assert.Contains(t, str, "test error")
}

// Test with different types
func TestResult_DifferentTypes(t *testing.T) {
	// String type
	strResult := Ok("hello")
	assert.Equal(t, "hello", strResult.Unwrap())

	// Slice type
	sliceResult := Ok([]int{1, 2, 3})
	assert.Equal(t, []int{1, 2, 3}, sliceResult.Unwrap())

	// Map type
	mapResult := Ok(map[string]int{"a": 1, "b": 2})
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, mapResult.Unwrap())

	// Struct type
	type TestStruct struct {
		Field1 string
		Field2 int
	}
	structResult := Ok(TestStruct{Field1: "test", Field2: 42})
	assert.Equal(t, "test", structResult.Unwrap().Field1)
	assert.Equal(t, 42, structResult.Unwrap().Field2)
}

// Test chaining multiple operations
func TestResult_ChainedOperations(t *testing.T) {
	// Chain multiple successful operations
	step1 := MapResult(Ok(5), func(x int) int { return x * 2 }) // 10
	step2 := AndThen(step1, func(x int) Result[int] {
		return Ok(x + 3) // 13
	})
	step3 := MapResult(step2, func(x int) int { return x * 10 }) // 130
	result := AndThen(step3, func(x int) Result[string] {
		return Ok(fmt.Sprintf("result: %d", x)) // "result: 130"
	})

	assert.True(t, result.IsOk())
	assert.Equal(t, "result: 130", result.Unwrap())
}

func TestResult_ChainedOperations_WithError(t *testing.T) {
	// Chain operations where one fails
	step1 := MapResult(Ok(5), func(x int) int { return x * 2 }) // 10
	step2 := AndThen(step1, func(x int) Result[int] {
		return Err[int](errors.New("operation failed")) // Error here
	})
	step3 := MapResult(step2, func(x int) int { return x * 10 }) // Should not execute
	result := AndThen(step3, func(x int) Result[string] {
		return Ok(fmt.Sprintf("result: %d", x)) // Should not execute
	})

	assert.True(t, result.IsErr())
	assert.Equal(t, "operation failed", result.Error().Error())
}

// Test error handling patterns
func TestResult_ErrorHandlingPatterns(t *testing.T) {
	t.Run("early_return_pattern", func(t *testing.T) {
		processValue := func(input string) Result[int] {
			if input == "" {
				return Err[int](errors.New("empty input"))
			}
			if input == "invalid" {
				return Err[int](errors.New("invalid input"))
			}
			return Ok(len(input))
		}

		// Test successful case
		result := processValue("hello")
		assert.True(t, result.IsOk())
		assert.Equal(t, 5, result.Unwrap())

		// Test error cases
		result = processValue("")
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "empty input")

		result = processValue("invalid")
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "invalid input")
	})

	t.Run("recovery_pattern", func(t *testing.T) {
		riskyOperation := func() Result[string] {
			return Err[string](errors.New("operation failed"))
		}

		result := riskyOperation().OrElse(func(err error) Result[string] {
			return Ok("default value")
		})

		assert.True(t, result.IsOk())
		assert.Equal(t, "default value", result.Unwrap())
	})
}

// Benchmark tests
func BenchmarkResult_Ok(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Ok(42)
	}
}

func BenchmarkResult_Err(b *testing.B) {
	err := errors.New("test error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Err[int](err)
	}
}

func BenchmarkResult_Map(b *testing.B) {
	result := Ok(42)
	mapper := func(x int) int { return x * 2 }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MapResult(result, mapper)
	}
}

func BenchmarkResult_ChainedOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		step1 := MapResult(Ok(5), func(x int) int { return x * 2 })
		step2 := AndThen(step1, func(x int) Result[int] { return Ok(x + 3) })
		MapResult(step2, func(x int) int { return x * 10 })
	}
}
