package goscript

import "fmt"

type basetest struct {
	X int
}

func (b *basetest) Foo() {
	fmt.Printf("Foo %v \n", b.X)
}

type test struct {
	basetest
	Y func(string)
}

func (t test) Bar() {
	fmt.Printf("Bar %v \n", t.X)
}
