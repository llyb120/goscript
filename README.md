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

- ~~不支持switch语句~~
- 不支持select语句
- 不支持泛型
- 不支持import语句
- ~~不支持go func()~~
- 不支持defer func()
- 不支持定义struct
- 不支持定义interface
- 无空指针异常，即使使用未定义的变量也不会出错
- 支持对go原生代码的桥接调用

## 与Go的互相调用

### 注册桥接函数

```go
res, err := interp.BindFunction("print", print)
if err != nil {
    // do something
}
fmt.Println(res)
```

### 绑定桥接对象

```go
// 绑定单一对象
interp.Set("x", 1)

// 绑定global对象
// 支持结构体和map
interp.SetGlobal(map[string]any{
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
a = a + 1 //will not panic
fmt.Println("hello" + a)
fmt.Println("world" + a.b)
var a strings.Builder
a.WriteString("hello")
print(a.String())

var b string
b = "hello" + "world"
print(b)

print("begin abc")
print(X)
print("bar")

Y("bar")

G.Bar()

yyy := func(a string){
	print("ok " + a)
}
yyy("foo")
mp := map[string]any{
	"x": 1,
	"y": 2,
}
mp = make(map[string]any)
mp["x"] = "foo"
yyy(mp.x)
mp["y"] = 4
mp.x = "foo jian"
var b = 1

yyy(mp.x)
	
sum := 0
j := 0
for j < 5 {
	print(j)
	j++
	if j > 2 {
		break	
	}
}
for i := 1; i <= 5; i++ {
	if i % 2 == 0 && 1 > 0 {
		sum += i * 2
		print(sum)
	} else {
		sum += i
		print(sum)
	}
}
return sum
```





