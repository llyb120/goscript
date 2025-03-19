package goscript

import (
	"fmt"
	"reflect"
	"strings"
)

// 注册标准库包
func (i *Interpreter) libs() {

	// strings 包
	i.Set("strings", map[string]any{
		"Builder": reflect.TypeOf(strings.Builder{}),
		"Join":    strings.Join,
		"Split":   strings.Split,
	})

	// fmt 包
	i.Set("fmt", map[string]any{
		"Println": fmt.Println,
		"Printf":  fmt.Printf,
		"Sprintf": fmt.Sprintf,
	})

	i.Set("len", func(v any) int {
		return reflect.ValueOf(v).Len()
	})
}
