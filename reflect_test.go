package goscript

import (
	"fmt"
	"testing"
)

type Person struct {
	Name  string
	Age   int
	Name2 *string
}

func (p Person) GetName() string {
	return p.Name
}

func (p *Person) SetName(name string) {
	p.Name = name
}

func TestReflect(t *testing.T) {
	cache := NewReflectCache()
	name := "李四"
	p := Person{Name: "张三", Age: 18, Name2: &name}

	// 获取字段
	if item, _ := cache.get(p, "Name"); item != nil {
		fmt.Println(item) // 输出: 张三
	}

	if item, _ := cache.get(p, "Name2"); item != nil {
		if ptr2, ok := item.(*string); ok {
			fmt.Println(*ptr2) // 输出: 李四
		}
	}

	// 获取方法
	if method, typ := cache.get(p, "SetName"); method != nil {
		fmt.Println(method)
		fmt.Printf("方法 SetName 是指针接收器方法: %v\n", typ)
		// 可以通过 method.Func.Call() 调用方法
	}
}

func TestReflectPtr(t *testing.T) {
	cache := NewReflectCache()
	name := "李四"
	p := &Person{Name: "张三", Age: 18, Name2: &name}

	// 获取字段
	if item, _ := cache.get(p, "Name"); item != nil {
		fmt.Println(item) // 输出: 张三
	}

	if item, _ := cache.get(p, "Name2"); item != nil {
		if ptr2, ok := item.(*string); ok {
			fmt.Println(*ptr2) // 输出: 李四
		}
	}

	// 获取方法
	if method, typ := cache.get(p, "SetName"); method != nil {
		fmt.Println(method)
		fmt.Printf("方法 SetName 是指针接收器方法: %v\n", typ)
		// 可以通过 method.Func.Call() 调用方法
	}
}
