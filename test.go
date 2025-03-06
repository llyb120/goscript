package goscript

import "fmt"

type basetest struct {
	X int
}

type basetest2 struct {
	YYY int
}

func (b *basetest) Foo() {
	fmt.Printf("Foo %v \n", b.X)
}

type test struct {
	basetest
	nnn int
	basetest2
	Y func(string)
}

func (t test) Bar() {
	fmt.Printf("Bar %v \n", t.X)
}

func (t test) TestArgs(a string, b ...string) {
	fmt.Printf("Bar2 %v \n", a)
	for _, v := range b {
		fmt.Printf("Bar2 %v \n", v)
	}
}
