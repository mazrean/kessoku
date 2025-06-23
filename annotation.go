// Package kessoku provides annotation-based dependency injection code generation for Go.
package kessoku

type name string

type provider interface {
	provide()
}

type fnProvider[T any] struct {
	fn T
}

func (p fnProvider[T]) provide() {}

func (p fnProvider[T]) Fn() T {
	return p.fn
}

func Provide[T any](fn T) fnProvider[T] {
	return fnProvider[T]{fn: fn}
}

type bindProvider[S, T any] fnProvider[T]

func (p bindProvider[_, _]) provide() {}

func (p bindProvider[_, T]) Fn() T {
	return p.fn
}

func Bind[S, T any](fn fnProvider[T]) bindProvider[S, T] {
	return bindProvider[S, T](fn)
}

func Value[T any](v T) fnProvider[func() T] {
	return fnProvider[func() T]{
		fn: func() T { return v },
	}
}

type argProvider[T any] struct{}

func (p argProvider[T]) provide() {}

func Arg[T any](name name) argProvider[T] {
	return argProvider[T]{}
}

func Inject[T any](name name, providers ...provider) struct{} {
	// dummy return
	return struct{}{}
}
