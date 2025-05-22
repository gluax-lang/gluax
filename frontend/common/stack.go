package common

import "iter"

// Stack is a generic LIFO container.
// Zero-value ready: just declare var s stack.Stack[int] and use it.
type Stack[T any] struct {
	items []T
}

// New returns an empty stack.
// Usage: s := stack.New[int]()
func New[T any]() *Stack[T] { return &Stack[T]{} }

// Push adds v to the top of the stack.
func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

// Pop removes and returns the top element.
// The bool result is false when the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	idx := len(s.items) - 1
	v := s.items[idx]
	// Avoid memory leak for large reference types
	s.items[idx] = zero
	s.items = s.items[:idx]
	return v, true
}

// Peek returns the top element without removing it.
func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Len returns the number of elements currently in the stack.
func (s *Stack[T]) Len() int { return len(s.items) }

// Empty reports whether the stack has no elements.
func (s *Stack[T]) Empty() bool { return len(s.items) == 0 }

// Clear discards all contents in O(1) time.
func (s *Stack[T]) Clear() {
	// help GC
	var zero T
	for i := range s.items {
		s.items[i] = zero
	}
	s.items = s.items[:0]
}

func (s *Stack[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.items {
			if !yield(v) {
				return
			}
		}
	}
}
