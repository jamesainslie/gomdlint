package functional

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSome(t *testing.T) {
	opt := Some(42)
	assert.True(t, opt.IsSome())
	assert.False(t, opt.IsNone())
	assert.Equal(t, 42, opt.Unwrap())
}

func TestNone(t *testing.T) {
	opt := None[int]()
	assert.False(t, opt.IsSome())
	assert.True(t, opt.IsNone())
}

func TestOption_Unwrap_Some(t *testing.T) {
	opt := Some("hello")
	value := opt.Unwrap()
	assert.Equal(t, "hello", value)
}

func TestOption_Unwrap_None_Panics(t *testing.T) {
	opt := None[string]()
	assert.Panics(t, func() {
		opt.Unwrap()
	})
}

func TestOption_UnwrapOr(t *testing.T) {
	// Test with Some value
	someOpt := Some(42)
	assert.Equal(t, 42, someOpt.UnwrapOr(100))

	// Test with None value
	noneOpt := None[int]()
	assert.Equal(t, 100, noneOpt.UnwrapOr(100))
}

func TestOption_UnwrapOrElse(t *testing.T) {
	fallback := func() int {
		return 999
	}

	// Test with Some value
	someOpt := Some(42)
	assert.Equal(t, 42, someOpt.UnwrapOrElse(fallback))

	// Test with None value
	noneOpt := None[int]()
	assert.Equal(t, 999, noneOpt.UnwrapOrElse(fallback))
}

func TestOption_Map(t *testing.T) {
	// Test mapping Some value
	someOpt := Some(5)
	mapped := MapOption(someOpt, func(x int) string {
		return fmt.Sprintf("value: %d", x)
	})
	assert.True(t, mapped.IsSome())
	assert.Equal(t, "value: 5", mapped.Unwrap())

	// Test mapping None value (should remain None)
	noneOpt := None[int]()
	mapped = MapOption(noneOpt, func(x int) string {
		return fmt.Sprintf("value: %d", x)
	})
	assert.True(t, mapped.IsNone())
}

func TestOption_FlatMap(t *testing.T) {
	// Test flat mapping Some value that returns Some
	someOpt := Some(5)
	flatMapped := FlatMap(someOpt, func(x int) Option[string] {
		return Some(fmt.Sprintf("value: %d", x))
	})
	assert.True(t, flatMapped.IsSome())
	assert.Equal(t, "value: 5", flatMapped.Unwrap())

	// Test flat mapping Some value that returns None
	flatMapped = FlatMap(someOpt, func(x int) Option[string] {
		return None[string]()
	})
	assert.True(t, flatMapped.IsNone())

	// Test flat mapping None value (should remain None)
	noneOpt := None[int]()
	flatMapped = FlatMap(noneOpt, func(x int) Option[string] {
		return Some("should not be called")
	})
	assert.True(t, flatMapped.IsNone())
}

func TestOption_Filter(t *testing.T) {
	// Test filtering Some value that passes predicate
	someOpt := Some(10)
	filtered := someOpt.Filter(func(x int) bool {
		return x > 5
	})
	assert.True(t, filtered.IsSome())
	assert.Equal(t, 10, filtered.Unwrap())

	// Test filtering Some value that fails predicate
	filtered = someOpt.Filter(func(x int) bool {
		return x > 15
	})
	assert.True(t, filtered.IsNone())

	// Test filtering None value (should remain None)
	noneOpt := None[int]()
	filtered = noneOpt.Filter(func(x int) bool {
		return true
	})
	assert.True(t, filtered.IsNone())
}

func TestOption_AndThen(t *testing.T) {
	// Test chaining Some values
	opt1 := Some(5)
	opt2 := FlatMap(opt1, func(x int) Option[int] {
		return Some(x * 2)
	})
	assert.True(t, opt2.IsSome())
	assert.Equal(t, 10, opt2.Unwrap())

	// Test chaining where first is Some, second is None
	opt2 = FlatMap(opt1, func(x int) Option[int] {
		return None[int]()
	})
	assert.True(t, opt2.IsNone())

	// Test chaining where first is None
	noneOpt := None[int]()
	opt2 = FlatMap(noneOpt, func(x int) Option[int] {
		return Some(x * 2)
	})
	assert.True(t, opt2.IsNone())
}

func TestOption_OrElse(t *testing.T) {
	// Test with Some value (should return original)
	someOpt := Some(42)
	result := someOpt.OrElse(func() Option[int] {
		return Some(100)
	})
	assert.True(t, result.IsSome())
	assert.Equal(t, 42, result.Unwrap())

	// Test with None value that recovers
	noneOpt := None[int]()
	result = noneOpt.OrElse(func() Option[int] {
		return Some(100)
	})
	assert.True(t, result.IsSome())
	assert.Equal(t, 100, result.Unwrap())

	// Test with None value that returns another None
	result = noneOpt.OrElse(func() Option[int] {
		return None[int]()
	})
	assert.True(t, result.IsNone())
}

func TestOption_String(t *testing.T) {
	// Test Some option string representation
	someOpt := Some(42)
	str := someOpt.String()
	assert.Contains(t, str, "Some")
	assert.Contains(t, str, "42")

	// Test None option string representation
	noneOpt := None[int]()
	str = noneOpt.String()
	assert.Contains(t, str, "None")
}

// Test with different types
func TestOption_DifferentTypes(t *testing.T) {
	// String type
	strOpt := Some("hello")
	assert.Equal(t, "hello", strOpt.Unwrap())

	// Slice type
	sliceOpt := Some([]int{1, 2, 3})
	assert.Equal(t, []int{1, 2, 3}, sliceOpt.Unwrap())

	// Map type
	mapOpt := Some(map[string]int{"a": 1, "b": 2})
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, mapOpt.Unwrap())

	// Pointer type
	value := 42
	ptrOpt := Some(&value)
	assert.Equal(t, &value, ptrOpt.Unwrap())
	assert.Equal(t, 42, *ptrOpt.Unwrap())

	// Struct type
	type TestStruct struct {
		Field1 string
		Field2 int
	}
	structOpt := Some(TestStruct{Field1: "test", Field2: 42})
	assert.Equal(t, "test", structOpt.Unwrap().Field1)
	assert.Equal(t, 42, structOpt.Unwrap().Field2)
}

// Test chaining multiple operations
func TestOption_ChainedOperations(t *testing.T) {
	// Chain multiple successful operations
	step1 := MapOption(Some(5), func(x int) int { return x * 2 }) // Some(10)
	step2 := FlatMap(step1, func(x int) Option[int] {
		return Some(x + 3) // Some(13)
	})
	step3 := MapOption(step2, func(x int) int { return x * 10 }) // Some(130)
	result := FlatMap(step3, func(x int) Option[string] {
		return Some(fmt.Sprintf("result: %d", x)) // Some("result: 130")
	})

	assert.True(t, result.IsSome())
	assert.Equal(t, "result: 130", result.Unwrap())
}

func TestOption_ChainedOperations_WithNone(t *testing.T) {
	// Chain operations where one returns None
	step1 := MapOption(Some(5), func(x int) int { return x * 2 }) // Some(10)
	step2 := FlatMap(step1, func(x int) Option[int] {
		return None[int]() // None here
	})
	step3 := MapOption(step2, func(x int) int { return x * 10 }) // Should not execute
	result := FlatMap(step3, func(x int) Option[string] {
		return Some(fmt.Sprintf("result: %d", x)) // Should not execute
	})

	assert.True(t, result.IsNone())
}

// Test common patterns
func TestOption_CommonPatterns(t *testing.T) {
	t.Run("safe_division", func(t *testing.T) {
		safeDivide := func(a, b int) Option[int] {
			if b == 0 {
				return None[int]()
			}
			return Some(a / b)
		}

		// Test successful division
		result := safeDivide(10, 2)
		assert.True(t, result.IsSome())
		assert.Equal(t, 5, result.Unwrap())

		// Test division by zero
		result = safeDivide(10, 0)
		assert.True(t, result.IsNone())
	})

	t.Run("safe_array_access", func(t *testing.T) {
		safeGet := func(arr []string, index int) Option[string] {
			if index < 0 || index >= len(arr) {
				return None[string]()
			}
			return Some(arr[index])
		}

		arr := []string{"a", "b", "c"}

		// Test valid index
		result := safeGet(arr, 1)
		assert.True(t, result.IsSome())
		assert.Equal(t, "b", result.Unwrap())

		// Test invalid index
		result = safeGet(arr, 5)
		assert.True(t, result.IsNone())

		// Test negative index
		result = safeGet(arr, -1)
		assert.True(t, result.IsNone())
	})

	t.Run("safe_map_access", func(t *testing.T) {
		safeMapGet := func(m map[string]int, key string) Option[int] {
			if value, exists := m[key]; exists {
				return Some(value)
			}
			return None[int]()
		}

		m := map[string]int{"a": 1, "b": 2}

		// Test existing key
		result := safeMapGet(m, "a")
		assert.True(t, result.IsSome())
		assert.Equal(t, 1, result.Unwrap())

		// Test non-existing key
		result = safeMapGet(m, "c")
		assert.True(t, result.IsNone())
	})

	t.Run("chained_safe_operations", func(t *testing.T) {
		parseAndDouble := func(s string) Option[int] {
			// Simplified parsing - just check if it's a known value
			switch s {
			case "5":
				return Some(5)
			case "10":
				return Some(10)
			default:
				return None[int]()
			}
		}

		safeDivide := func(a, b int) Option[int] {
			if b == 0 {
				return None[int]()
			}
			return Some(a / b)
		}

		// Chain successful operations
		step1 := parseAndDouble("10")
		step2 := FlatMap(step1, func(x int) Option[int] {
			return safeDivide(x, 2)
		})
		result := MapOption(step2, func(x int) int {
			return x * 3
		})

		assert.True(t, result.IsSome())
		assert.Equal(t, 15, result.Unwrap()) // 10 / 2 * 3 = 15

		// Chain where parsing fails
		step1 = parseAndDouble("invalid")
		step2 = FlatMap(step1, func(x int) Option[int] {
			return safeDivide(x, 2)
		})
		result = MapOption(step2, func(x int) int {
			return x * 3
		})

		assert.True(t, result.IsNone())
	})
}

// Test edge cases
func TestOption_EdgeCases(t *testing.T) {
	t.Run("nil_pointer", func(t *testing.T) {
		var ptr *int = nil
		opt := Some(ptr)
		assert.True(t, opt.IsSome())
		assert.Nil(t, opt.Unwrap())
	})

	t.Run("zero_values", func(t *testing.T) {
		// Zero int
		intOpt := Some(0)
		assert.True(t, intOpt.IsSome())
		assert.Equal(t, 0, intOpt.Unwrap())

		// Empty string
		strOpt := Some("")
		assert.True(t, strOpt.IsSome())
		assert.Equal(t, "", strOpt.Unwrap())

		// Nil slice
		var slice []int = nil
		sliceOpt := Some(slice)
		assert.True(t, sliceOpt.IsSome())
		assert.Nil(t, sliceOpt.Unwrap())

		// Empty slice
		emptySlice := []int{}
		sliceOpt = Some(emptySlice)
		assert.True(t, sliceOpt.IsSome())
		assert.Equal(t, []int{}, sliceOpt.Unwrap())
	})
}

// Benchmark tests
func BenchmarkOption_Some(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Some(42)
	}
}

func BenchmarkOption_None(b *testing.B) {
	for i := 0; i < b.N; i++ {
		None[int]()
	}
}

func BenchmarkOption_Map(b *testing.B) {
	opt := Some(42)
	mapper := func(x int) int { return x * 2 }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MapOption(opt, mapper)
	}
}

func BenchmarkOption_ChainedOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		step1 := MapOption(Some(5), func(x int) int { return x * 2 })
		step2 := FlatMap(step1, func(x int) Option[int] { return Some(x + 3) })
		MapOption(step2, func(x int) int { return x * 10 })
	}
}
