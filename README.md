# GoScript

GoScript 是一个用 Go 语言编写的Go脚本解释器。

## 快速上手

```go
package main

func main() {
	interp := NewInterpreter()
    interp.Interpret(`print("Hello, World!")`)
}
```

## 语法
作为一个脚本语言，有其特定的应用场景

语法对go进行兼容，支持go的绝大部分语法，但在此基础上进行精简，与go的差异如下

- 不支持switch语句
- 不支持select语句
- 不支持泛型
- 不支持import语句
- 不支持go func()
- 不支持defer func()
- 不支持定义struct
- 不支持定义interface

为了弥补上述缺陷，支持对go原生代码的桥接调用

## 与Go的互相调用

### 注册桥接函数

```go
res, err := interp.RegisterFunction("print", print)
if err != nil {
    // do something
}
fmt.Println(res)
```

### 绑定桥接对象

```go
// 绑定单一对象
interp.BindObject("x", 1)

// 绑定global对象
// 支持结构体和map
interp.BindGlobalObject(map[string]any{
    "x": 1,
    "y": func(s string) {
        fmt.Printf("fomat by y   %v \n", s)
    },
})

type test struct {
	X int
	Y func(string)
}

func (t *test) Bar() {
	fmt.Println("Bar")
}

interp.BindGlobalObject(test{
	X: 222,
	Y: func(s string) {
		fmt.Printf("fomat by y   %v \n", s)
	},
})
```

脚本中可以直接调用global对象上的属性，或者使用G关键字调用global对象

```go
// script
print(x) //will print 1
y("hello") //will print fomat by y   hello
G.y("hello") //will print fomat by y   hello
G.Bar() //will print Bar
```





