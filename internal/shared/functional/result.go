package functional

import (
	"encoding/json"
	"fmt"
)

// Result represents the result of an operation that can either succeed with a value or fail with an error.
// This follows functional programming principles for explicit error handling.
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful Result with the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value, err: nil}
}

// Err creates a failed Result with the given error.
func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{value: zero, err: err}
}

// IsOk returns true if the Result contains a value.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the Result contains an error.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Unwrap returns the contained value or panics if the Result is an error.
// Use with caution - prefer UnwrapOr or AndThen for safe operations.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(fmt.Sprintf("called Unwrap on Err: %v", r.err))
	}
	return r.value
}

// UnwrapOr returns the contained value or the provided default if error.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return r.value
}

// UnwrapOrElse returns the contained value or computes it from the provided function.
func (r Result[T]) UnwrapOrElse(fn func(error) T) T {
	if r.err != nil {
		return fn(r.err)
	}
	return r.value
}

// Error returns the contained error or nil if successful.
func (r Result[T]) Error() error {
	return r.err
}

// Value returns the contained value and error (similar to Go's standard pattern).
func (r Result[T]) Value() (T, error) {
	return r.value, r.err
}

// MapResult transforms the contained value using the provided function.
// If the Result is an error, returns the error without calling the function.
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}

// MapErr transforms the contained error using the provided function.
// If the Result is successful, returns the success without calling the function.
func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err != nil {
		return Err[T](fn(r.err))
	}
	return r
}

// AndThen (flatMap/bind) chains operations that return Results.
// If the current Result is an error, returns the error without calling the function.
func AndThen[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return fn(r.value)
}

// OrElse returns this Result if it's successful, otherwise returns the alternative.
func (r Result[T]) OrElse(alternative Result[T]) Result[T] {
	if r.err != nil {
		return alternative
	}
	return r
}

// Filter returns the Result if the predicate is satisfied, otherwise returns an error.
func (r Result[T]) Filter(predicate func(T) bool, errFn func(T) error) Result[T] {
	if r.err != nil {
		return r
	}
	if !predicate(r.value) {
		return Err[T](errFn(r.value))
	}
	return r
}

// String implements the Stringer interface for better debugging.
func (r Result[T]) String() string {
	if r.err != nil {
		return fmt.Sprintf("Err(%v)", r.err)
	}
	return fmt.Sprintf("Ok(%v)", r.value)
}

// MarshalJSON implements json.Marshaler.
func (r Result[T]) MarshalJSON() ([]byte, error) {
	if r.err != nil {
		return json.Marshal(map[string]interface{}{
			"error": r.err.Error(),
		})
	}
	return json.Marshal(map[string]interface{}{
		"value": r.value,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Result[T]) UnmarshalJSON(data []byte) error {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if errStr, hasErr := obj["error"]; hasErr {
		*r = Err[T](fmt.Errorf("%v", errStr))
		return nil
	}

	if valueData, hasValue := obj["value"]; hasValue {
		valueBytes, err := json.Marshal(valueData)
		if err != nil {
			return err
		}

		var value T
		if err := json.Unmarshal(valueBytes, &value); err != nil {
			return err
		}

		*r = Ok(value)
		return nil
	}

	*r = Err[T](fmt.Errorf("invalid Result JSON structure"))
	return nil
}

// ToOption converts the Result to an Option (Some if Ok, None if Err).
func (r Result[T]) ToOption() Option[T] {
	if r.err != nil {
		return None[T]()
	}
	return Some(r.value)
}

// FromValue creates a Result from a value and error (standard Go pattern).
func FromValue[T any](value T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(value)
}

// TryFrom creates a Result from a function that follows Go's error convention.
func TryFrom[T any](fn func() (T, error)) Result[T] {
	value, err := fn()
	return FromValue(value, err)
}

// Collect transforms a slice of Results into a Result of slice.
// If any Result is an error, returns the first error encountered.
func Collect[T any](results []Result[T]) Result[[]T] {
	values := make([]T, 0, len(results))
	for _, result := range results {
		if result.err != nil {
			return Err[[]T](result.err)
		}
		values = append(values, result.value)
	}
	return Ok(values)
}

// MapSlice applies a function to each element of a slice, returning a Result of the transformed slice.
func MapSlice[T, U any](slice []T, fn func(T) Result[U]) Result[[]U] {
	results := make([]Result[U], len(slice))
	for i, item := range slice {
		results[i] = fn(item)
	}
	return Collect(results)
}
