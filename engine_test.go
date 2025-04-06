package goscript

import (
	"fmt"
	"testing"
)

func TestInterpreter(t *testing.T) {
	main()
}

type A struct {
}

func (a A) Test() string {
	return "base"
}

type B struct {
	A
}

func (b *B) Test() string {
	return "override"
}

func TestInterpreterOverride(t *testing.T) {
	interp := NewInterpreter()

	// 注册内置函数
	interp.Set("print", func(s any) {
		fmt.Printf("%v \n", s)
	})

	interp.SetGlobal(&B{})
	interp.Interpret(`
print(Test()) // 输出: override
`)
}
