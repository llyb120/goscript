package goscript

import (
	"fmt"
	"testing"
)

func TestString(t *testing.T) {
	interp := NewInterpreter()
	interp.Interpret(`
		print('Hello, World \'1\' 23424 !')
		print("Hello, World 'hohoafa' 23424 !")

		fn := func()(ret string) {
			ret = "123"
		}

		print(fn())
	`)

	fmt.Println(123)
}

// // 测试解释器对单引号字符串的处理
// func TestSingleQuoteString(t *testing.T) {
// 	interp := NewInterpreter()

// 	t.Run("Simple", func(t *testing.T) {
// 		res, err := interp.Interpret(`'hello'`)
// 		if err != nil {
// 			t.Fatalf("interpret error: %v", err)
// 		}
// 		if res != "hello" {
// 			t.Fatalf("expected hello, got %v", res)
// 		}
// 	})

// 	t.Run("WithEscapedSingleQuote", func(t *testing.T) {
// 		code := `'Hello, World \'1\' 23424 !'`
// 		expected := "Hello, World '1' 23424 !"
// 		res, err := interp.Interpret(code)
// 		if err != nil {
// 			t.Fatalf("interpret error: %v", err)
// 		}
// 		if res != expected {
// 			t.Fatalf("expected %q, got %v", expected, res)
// 		}
// 	})

// 	t.Run("MixedQuotes", func(t *testing.T) {
// 		code := `print('He said "hi"')`
// 		expected := "He said \"hi\""
// 		res, err := interp.Interpret(code)
// 		if err != nil {
// 			t.Fatalf("interpret error: %v", err)
// 		}
// 		if res != expected {
// 			t.Fatalf("expected %q, got %v", expected, res)
// 		}
// 	})

// 	t.Run("AssignmentAndUsage", func(t *testing.T) {
// 		res, err := interp.Interpret(`x := 'foo'; x`)
// 		if err != nil {
// 			t.Fatalf("interpret error: %v", err)
// 		}
// 		if res != "foo" {
// 			t.Fatalf("expected foo, got %v", res)
// 		}
// 	})
// }
