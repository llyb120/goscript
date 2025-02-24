package goscript

import "fmt"

type test struct {
	X int
	Y func(string)
}

func (t test) Bar() {
	fmt.Printf("Bar %v \n", t.X)
}
