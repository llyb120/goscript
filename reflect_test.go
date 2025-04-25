package goscript

import (
	"fmt"
	"testing"
)

type Person struct {
	Name  string
	Age   int
	Name2 *string
	test  int
}

func (p Person) GetName() string {
	return p.Name
}

func (p *Person) SetName(name string) {
	p.Name = name
}

func (p *Person) doTest() int {
	fmt.Println("doTest success")
	return 1
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

	if method, typ := cache.get(p, "doTest"); method != nil {
		fmt.Println(method)
		fmt.Printf("方法 doTest 是指针接收器方法: %v\n", typ)
	}

	func(pp any) {
		cache.set(&pp, "Name", "李四的儿子")
		fmt.Println(pp)
	}(p)

	// cache.set(&p, "Name", "李四的儿子")
	// fmt.Println(p)
}

func TestReflectPtr(t *testing.T) {
	cache := NewReflectCache()
	name := "李四"
	p := &Person{Name: "张三", Age: 18, Name2: &name, test: 1}

	if item, _ := cache.get(p, "test"); item != nil {
		fmt.Println(item)
	}

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

	cache.set(p, "Name", "李四的儿子")
	fmt.Println(p)
}
