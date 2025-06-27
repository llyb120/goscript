package goscript

import (
	"fmt"
	"testing"
)

type TestStruct struct {
	Name string
	Age  int
	City string
}

type EmbeddedStruct struct {
	TestStruct
	Country string
}

func TestHasFunction(t *testing.T) {
	interp := NewInterpreter()

	// 注册内置函数
	interp.Set("print", func(args ...any) {
		fmt.Println(args...)
	})

	// 测试数组包含检查
	t.Run("Array Contains", func(t *testing.T) {
		interp.Set("arr", []int{1, 2, 3, 4, 5})

		result, err := interp.Interpret(`has(arr, 3)`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		result, err = interp.Interpret(`has(arr, 6)`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// 测试字符串数组包含检查
	t.Run("String Array Contains", func(t *testing.T) {
		interp.Set("strArr", []string{"apple", "banana", "orange"})

		result, err := interp.Interpret(`has(strArr, "banana")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		result, err = interp.Interpret(`has(strArr, "grape")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// 测试map包含检查
	t.Run("Map Contains", func(t *testing.T) {
		testMap := map[string]int{
			"a": 1,
			"b": 2,
			"c": 3,
		}
		interp.Set("testMap", testMap)

		result, err := interp.Interpret(`has(testMap, "b", "a", "c")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		result, err = interp.Interpret(`has(testMap, "d")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// 测试结构体字段检查
	t.Run("Struct Field Check", func(t *testing.T) {
		testStruct := TestStruct{
			Name: "John",
			Age:  30,
			City: "New York",
		}
		interp.Set("testStruct", testStruct)

		result, err := interp.Interpret(`has(testStruct, "Name")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		result, err = interp.Interpret(`has(testStruct, "Age")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		result, err = interp.Interpret(`has(testStruct, "NonExistentField")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})

	// 测试嵌入结构体字段检查
	t.Run("Embedded Struct Field Check", func(t *testing.T) {
		embeddedStruct := EmbeddedStruct{
			TestStruct: TestStruct{
				Name: "Jane",
				Age:  25,
				City: "London",
			},
			Country: "UK",
		}
		interp.Set("embeddedStruct", embeddedStruct)

		// 检查嵌入结构体的字段
		result, err := interp.Interpret(`has(embeddedStruct, "Name")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}

		// 检查自身的字段
		result, err = interp.Interpret(`has(embeddedStruct, "Country")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
	})

	// 测试指针类型
	t.Run("Pointer Type Check", func(t *testing.T) {
		testStruct := &TestStruct{
			Name: "Bob",
			Age:  35,
			City: "Paris",
		}
		interp.Set("ptrStruct", testStruct)

		result, err := interp.Interpret(`has(ptrStruct, "Name")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
	})

	// 测试nil值
	t.Run("Nil Value Check", func(t *testing.T) {
		interp.Set("nilValue", nil)

		result, err := interp.Interpret(`has(nilValue, "anything")`)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
	})
}

func TestHasIntegration(t *testing.T) {
	interp := NewInterpreter()

	// 注册内置函数
	interp.Set("print", func(args ...any) {
		fmt.Println(args...)
	})

	// 设置测试数据
	interp.Set("numbers", []int{1, 2, 3, 4, 5})
	interp.Set("userMap", map[string]string{
		"name": "Alice",
		"role": "admin",
	})

	testUser := TestStruct{
		Name: "Charlie",
		Age:  28,
		City: "Tokyo",
	}
	interp.Set("user", testUser)

	// 综合测试
	_, err := interp.Interpret(`
		print("Testing has function:")
		print("numbers contains 3:", has(numbers, 3))
		print("numbers contains 6:", has(numbers, 6))
		print("userMap has 'name':", has(userMap, "name"))
		print("userMap has 'email':", has(userMap, "email"))
		print("user has 'Name':", has(user, "Name"))
		print("user has 'Email':", has(user, "Email"))
	`)

	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}
}
