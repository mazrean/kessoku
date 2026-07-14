package main

// A and B form a direct dependency cycle: A requires B, B requires A.

type A struct{ b *B }
type B struct{ a *A }

func NewA(b *B) *A { return &A{b: b} }
func NewB(a *A) *B { return &B{a: a} }

func main() {}
