package functional

import (
	"encoding/json"
	"fmt"
)

// Option represents an optional value using functional programming principles.
// It provides a safe way to handle potentially null values without nil pointer exceptions.
type Option[T any] struct {
	value *T
}

// Some creates an Option containing the given value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: &value}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{value: nil}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.value != nil
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return o.value == nil
}

// Unwrap returns the contained value or panics if None.
// Use with caution - prefer UnwrapOr or Map for safe operations.
func (o Option[T]) Unwrap() T {
	if o.value == nil {
		panic("called Unwrap on None")
	}
	return *o.value
}

// UnwrapOr returns the contained value or the provided default.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if o.value == nil {
		return defaultValue
	}
	return *o.value
}

// UnwrapOrElse returns the contained value or computes it from the provided function.
func (o Option[T]) UnwrapOrElse(fn func() T) T {
	if o.value == nil {
		return fn()
	}
	return *o.value
}

// MapOption transforms the contained value using the provided function.
// If None, returns None without calling the function.
func MapOption[T, U any](o Option[T], fn func(T) U) Option[U] {
	if o.value == nil {
		return None[U]()
	}
	return Some(fn(*o.value))
}

// FlatMap (bind/chain) transforms the contained value and flattens the result.
func FlatMap[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
	if o.value == nil {
		return None[U]()
	}
	return fn(*o.value)
}

// Filter returns the Option if the predicate is satisfied, otherwise None.
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
	if o.value == nil || !predicate(*o.value) {
		return None[T]()
	}
	return o
}

// OrElse returns this Option if it contains a value, otherwise calls the function to get an alternative.
func (o Option[T]) OrElse(fn func() Option[T]) Option[T] {
	if o.value != nil {
		return o
	}
	return fn()
}

// String implements the Stringer interface for better debugging.
func (o Option[T]) String() string {
	if o.value == nil {
		return "None"
	}
	if data, err := json.Marshal(*o.value); err == nil {
		return "Some(" + string(data) + ")"
	}
	return fmt.Sprintf("Some(%v)", *o.value)
}

// MarshalJSON implements json.Marshaler.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*o = Some(value)
	return nil
}

// ToSlice converts the Option to a slice (empty or single element).
func (o Option[T]) ToSlice() []T {
	if o.value == nil {
		return []T{}
	}
	return []T{*o.value}
}

// FromPointer creates an Option from a pointer (Some if non-nil, None if nil).
func FromPointer[T any](ptr *T) Option[T] {
	if ptr == nil {
		return None[T]()
	}
	return Some(*ptr)
}

// ToPointer converts the Option to a pointer (nil if None).
func (o Option[T]) ToPointer() *T {
	return o.value
}
