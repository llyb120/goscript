package goscript

import (
	"fmt"
	"reflect"
	"strings"
)

// 注册标准库包
func (i *Interpreter) registerStandardPackages() {

	// strings 包
	i.sharedScope.objects["strings"] = map[string]any{
		"Builder": reflect.TypeOf(strings.Builder{}),
		"Join":    strings.Join,
		"Split":   strings.Split,
	}

	// fmt 包
	i.sharedScope.objects["fmt"] = map[string]any{
		"Println": fmt.Println,
		"Printf":  fmt.Printf,
		"Sprintf": fmt.Sprintf,
	}
}
