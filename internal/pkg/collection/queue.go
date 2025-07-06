// Package collection provides utility data structures.
package collection

import (
	"container/list"
)

type Queue[T any] struct {
	data list.List
}

func NewQueue[T any]() *Queue[T] {
	return &Queue[T]{}
}

func (q *Queue[T]) Push(v T) {
	q.data.PushBack(v)
}

func (q *Queue[T]) Pop() T {
	e := q.data.Front()
	if e == nil {
		var zero T
		return zero
	}

	q.data.Remove(e)
	return e.Value.(T)
}

func (q *Queue[T]) Peek() T {
	e := q.data.Front()
	if e == nil {
		var zero T
		return zero
	}

	return e.Value.(T)
}

func (q *Queue[T]) Len() int {
	return q.data.Len()
}

func (q *Queue[T]) Iter(yield func(T) bool) {
	for e := q.data.Front(); e != nil; e = q.data.Front() {
		q.data.Remove(e)

		if !yield(e.Value.(T)) {
			break
		}
	}
}

// ToSlice returns all elements in the queue as a slice without modifying the queue
func (q *Queue[T]) ToSlice() []T {
	result := make([]T, 0, q.data.Len())
	for e := q.data.Front(); e != nil; e = e.Next() {
		result = append(result, e.Value.(T))
	}
	return result
}
