package goscript

import (
	"fmt"
	"reflect"
	"strings"
)

// 注册标准库包
func (i *Interpreter) registerStandardPackages() {
	// 注册 strings 包
	stringsPackage := make(map[string]any)

	// 注册 strings.Builder 类型
	builderType := reflect.TypeOf(strings.Builder{})
	stringsPackage["Builder"] = builderType

	// 注册 strings 包中的函数
	stringsPackage["Join"] = strings.Join
	stringsPackage["Split"] = strings.Split
	// 添加更多 strings 包函数...

	// 将包注册为全局作用域中的对象
	i.scope.objects["strings"] = stringsPackage

	// 注册 fmt 包
	i.scope.objects["fmt"] = map[string]any{
		"Println": fmt.Println,
		"Printf":  fmt.Printf,
		"Sprintf": fmt.Sprintf,
	}
}
