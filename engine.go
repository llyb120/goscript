package goscript

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

// 因为类型是有限的，所以可以做一个全局的缓存
var globalReflectCache = NewReflectCache()

type Function struct {
	params []*ast.Field
	body   *ast.BlockStmt
}

type Interpreter struct {
	// sharedScope *SharedScope
	scope    *Scope
	global   any
	astCache *astCache
	isForked bool
}

// func NewInterpreterWithSharedScope(sharedScope map[string]any) *Interpreter {
// 	interp := NewInterpreter()
// 	for k, v := range sharedScope {
// 		interp.sharedScope.Store(k, v)
// 	}
// 	return interp
// }

func NewInterpreter() *Interpreter {
	interp := &Interpreter{
		// sharedScope 只可读不可写
		// 只在初始化的时候给一次写入的机会
		// sharedScope: &SharedScope{},
		scope:  &Scope{},
		global: nil,
		astCache: &astCache{
			cache: make(map[string]*ast.File),
		},
	}

	// 注册标准库包作为全局作用域中的对象
	interp.libs()

	return interp
}

func (i *Interpreter) Fork() *Interpreter {
	globalScope := &Scope{}
	globalScope.parent = i.scope
	return &Interpreter{
		scope:  globalScope,
		global: i.global,
		// 共享
		astCache: i.astCache,
		isForked: true,
	}
}

// func (i *Interpreter) BindFunction(name string, fn any) {
// 	// 只有主进程可以绑定函数
// 	if i.isForked {
// 		return
// 	}
// 	if _, ok := i.sharedScope.Load(name); ok {
// 		return
// 	}
// 	i.sharedScope.Store(name, reflect.ValueOf(fn))
// }

func (i *Interpreter) Set(name string, obj any) {
	fnValue := reflect.ValueOf(obj)
	if fnValue.Kind() != reflect.Func {
		i.scope.Store(name, obj)
	} else {
		i.scope.Store(name, fnValue)
	}
}

func (i *Interpreter) SetGlobal(obj any) {
	refVal := reflect.ValueOf(obj)
	if refVal.Kind() == reflect.Struct {
		ptr := reflect.New(reflect.TypeOf(obj))
		ptr.Elem().Set(refVal)
		i.global = ptr.Interface()
	} else {
		i.global = obj
	}
}

func (i *Interpreter) GetGlobal() any {
	return i.global
}

func (i *Interpreter) Interpret(code string) (any, error) {
	fset := token.NewFileSet()
	code = `package main
	func __main__() any {	
	` + code + `
	}
	`
	// fmt.Println(code)
	astFile, err := i.astCache.GetIfNotExist(code, func() (*ast.File, error) {
		return parser.ParseFile(fset, "", code, parser.Mode(0))
	})
	if err != nil {
		return nil, err
	}
	return i.eval(astFile.Decls[0].(*ast.FuncDecl).Body)
	// 首先处理所有函数定义
	// for _, decl := range f.Decls {
	// 	if funcDecl, ok := decl.(*ast.FuncDecl); ok {
	// 		if funcDecl.Name.Name != "__main__" {
	// 			i.userFuncs[funcDecl.Name.Name] = &Function{
	// 				params: funcDecl.Type.Params.List,
	// 				body:   funcDecl.Body,
	// 			}
	// 		}
	// 	}
	// }

	// // 查找并执行 __main__ 函数
	// for _, decl := range f.Decls {
	// 	if funcDecl, ok := decl.(*ast.FuncDecl); ok {
	// 		if funcDecl.Name.Name == "__main__" {
	// 			return i.eval(funcDecl.Body)
	// 		}
	// 	}
	// }
	// return nil, fmt.Errorf("没有找到 __main__ 函数")
}

func (i *Interpreter) eval(node ast.Node) (any, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return i.evalBasicLit(n)
	case *ast.Ident:
		return i.evalIdent(n)
	case *ast.BinaryExpr:
		return i.evalBinaryExpr(n)
	case *ast.CallExpr:
		// 处理 make 内置函数
		if ident, ok := n.Fun.(*ast.Ident); ok && ident.Name == "make" {
			if len(n.Args) == 0 {
				return nil, fmt.Errorf("make 需要至少一个参数")
			}

			switch t := n.Args[0].(type) {
			case *ast.MapType:
				// 创建新的 map
				return make(map[string]any), nil
			case *ast.ArrayType:
				// 创建新的 slice
				size := 0
				if len(n.Args) > 1 {
					// 如果提供了大小参数
					sizeVal, err := i.eval(n.Args[1])
					if err != nil {
						return nil, err
					}
					if sizeInt, ok := sizeVal.(int); ok {
						size = sizeInt
					}
				}
				return make([]any, size), nil
			default:
				return nil, fmt.Errorf("不支持的 make 类型: %T", t)
			}
		}
		return i.evalCallExpr(n)
	case *ast.ParenExpr:
		return i.eval(n.X)
	case *ast.BlockStmt:
		return i.evalBlockStmt(n)
	case *ast.ForStmt:
		return i.evalForStmt(n)
	case *ast.IfStmt:
		return i.evalIfStmt(n)
	case *ast.AssignStmt:
		return i.evalAssignStmt(n)
	case *ast.ReturnStmt:
		return i.evalReturnStmt(n)
	case *ast.IncDecStmt:
		return i.evalIncDecStmt(n)
	case *ast.ExprStmt:
		return i.eval(n.X)
	case *ast.FuncLit:
		return i.evalFuncLit(n)
	case *ast.CompositeLit:
		return i.evalCompositeLit(n)
	case *ast.KeyValueExpr:
		return i.evalKeyValueExpr(n)
	case *ast.IndexExpr:
		return i.evalIndexExpr(n)
	case *ast.SelectorExpr:
		return i.evalSelectorExpr(n)
	case *ast.DeclStmt:
		return i.evalDeclStmt(n)
	case *ast.MapType:
		// 直接支持 map 类型
		return make(map[string]any), nil
	case *ast.BranchStmt:
		return i.evalBranchStmt(n)
	case *ast.RangeStmt:
		return i.evalRangeStmt(n)
	case *ast.UnaryExpr:
		return i.evalUnaryExpr(n)
	case *ast.SwitchStmt:
		return i.evalSwitchStmt(n)
	default:
		return nil, fmt.Errorf("unsupported node type: %T", node)
	}
}

// 基础类型处理
func (i *Interpreter) evalBasicLit(lit *ast.BasicLit) (any, error) {
	switch lit.Kind {
	case token.INT:
		return strconv.Atoi(lit.Value)
	case token.STRING:
		// 处理字符串字面量，支持双引号和反引号
		if strings.HasPrefix(lit.Value, "`") && strings.HasSuffix(lit.Value, "`") {
			// 反引号字符串，直接去掉首尾的反引号
			return lit.Value[1 : len(lit.Value)-1], nil
		}
		// 双引号字符串，去掉首尾的双引号
		return strings.Trim(lit.Value, `"`), nil
	case token.FLOAT:
		return strconv.ParseFloat(lit.Value, 64)
	default:
		return nil, fmt.Errorf("unsupported literal type: %s", lit.Kind)
	}
}

// 处理标识符（变量/常量）
func (i *Interpreter) evalIdent(ident *ast.Ident) (any, error) {
	// 先检查是否是预定义常量
	switch ident.Name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil
	case "G":
		return i.global, nil
	}

	// 作用域链查找
	currentScope := i.scope
	for currentScope != nil {
		if val, ok := currentScope.Load(ident.Name); ok {
			return val, nil
		}
		currentScope = currentScope.parent
	}

	// 尝试从 __global__ 中获取
	if i.global != nil {
		switch g := i.global.(type) {
		case map[string]any:
			if val, exists := g[ident.Name]; exists {
				return val, nil
			}
		default:
			v := reflect.ValueOf(g)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			// 如果是map
			// if v.Kind() == reflect.Map {
			// 	return v.MapIndex(reflect.ValueOf(ident.Name)).Interface(), nil
			// }
			// 如果是slice
			if v.Kind() == reflect.Struct {
				if item, _ := globalReflectCache.get(g, ident.Name); item != nil {
					return item, nil
				}
				// if field := v.FieldByName(ident.Name); field.IsValid() {
				// 	// 检查字段是否可导出（首字母大写）
				// 	if field.CanInterface() {
				// 		return field.Interface(), nil
				// 	}
				// 	// 对于私有字段，返回错误
				// 	return nil, fmt.Errorf("无法访问私有字段: %s", ident.Name)
				// }
				// // 尝试查找方法
				// if method := v.MethodByName(ident.Name); method.IsValid() {
				// 	if method.CanInterface() {
				// 		return method.Interface(), nil
				// 	}
				// 	return nil, fmt.Errorf("无法访问私有方法: %s", ident.Name)
				// }
				// // 如果是指针，也查找指针的方法
				// if v.CanAddr() {
				// 	if method := v.Addr().MethodByName(ident.Name); method.IsValid() {
				// 		return method.Interface(), nil
				// 	}
				// }
			}
		}
	}

	fmt.Println("warn: 未定义的标识符: ", ident.Name)
	return nil, nil
	// return nil, fmt.Errorf("未定义的标识符: %s", ident.Name)
}

// 处理代码块
func (i *Interpreter) evalBlockStmt(block *ast.BlockStmt) (any, error) {
	// 创建新的作用域
	prevScope := i.scope
	i.scope = &Scope{
		parent: prevScope,
	}
	defer func() { i.scope = prevScope }() // 确保作用域恢复

	var result any
	var err error

	for _, stmt := range block.List {
		result, err = i.eval(stmt)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// 处理for循环
func (i *Interpreter) evalForStmt(f *ast.ForStmt) (any, error) {
	// 初始化语句
	if f.Init != nil {
		_, err := i.eval(f.Init)
		if err != nil {
			return nil, err
		}
	}

	for {
		// 检查终止条件
		if f.Cond != nil {
			cond, err := i.eval(f.Cond)
			if err != nil {
				return nil, err
			}
			if !toBool(cond) {
				break
			}
		}

		// 执行循环体
		result, err := i.eval(f.Body)
		if err != nil {
			return nil, err
		}

		// 处理 break 和 continue
		switch result.(type) {
		case breakSentinel:
			return nil, nil
		case continueSentinel:
			// 跳过后续处理，直接进入下一次循环
			if f.Post != nil {
				_, err := i.eval(f.Post)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		// 执行后续操作
		if f.Post != nil {
			_, err := i.eval(f.Post)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// 处理if语句
func (i *Interpreter) evalIfStmt(ifStmt *ast.IfStmt) (any, error) {
	// 初始化语句（如 if x := ...; x > 0 {}）
	if ifStmt.Init != nil {
		_, err := i.eval(ifStmt.Init)
		if err != nil {
			return nil, err
		}
	}

	// 评估条件
	cond, err := i.eval(ifStmt.Cond)
	if err != nil {
		return nil, err
	}

	if toBool(cond) {
		return i.eval(ifStmt.Body)
	} else if ifStmt.Else != nil {
		return i.eval(ifStmt.Else)
	}
	return nil, nil
}

// 处理赋值语句
func (i *Interpreter) evalAssignStmt(assign *ast.AssignStmt) (any, error) {
	// 处理右侧表达式
	values := make([]any, len(assign.Rhs))
	for idx, expr := range assign.Rhs {
		val, err := i.eval(expr)
		if err != nil {
			return nil, err
		}
		values[idx] = val
	}

	switch assign.Tok {
	case token.DEFINE, token.ASSIGN: // := 或 =
		for idx, lhs := range assign.Lhs {
			switch l := lhs.(type) {
			case *ast.Ident:
				if assign.Tok == token.DEFINE {
					i.scope.Store(l.Name, values[idx])
				} else {
					// 查找变量并赋值
					currentScope := i.scope
					for currentScope != nil {
						if _, ok := currentScope.Load(l.Name); ok {
							currentScope.Store(l.Name, values[idx])
							break
						}
						currentScope = currentScope.parent
					}
				}
			case *ast.IndexExpr:
				// 获取容器
				container, err := i.eval(l.X)
				if err != nil {
					return nil, err
				}

				// 获取索引
				index, err := i.eval(l.Index)
				if err != nil {
					return nil, err
				}

				// 根据容器类型进行赋值
				switch c := container.(type) {
				case map[any]any:
					c[index] = values[idx]
				case map[string]any:
					if strKey, ok := index.(string); ok {
						c[strKey] = values[idx]
					} else {
						return nil, fmt.Errorf("map键必须是字符串类型")
					}
				case []any:
					if intIndex, ok := index.(int); ok {
						if intIndex < 0 || intIndex >= len(c) {
							return nil, fmt.Errorf("索引越界")
						}
						c[intIndex] = values[idx]
					} else {
						return nil, fmt.Errorf("slice索引必须是整数")
					}
				default:
					return nil, fmt.Errorf("不支持的索引赋值操作: %T", container)
				}
			case *ast.SelectorExpr:
				// 获取容器
				container, err := i.eval(l.X)
				if err != nil {
					return nil, err
				}

				// 根据容器类型进行赋值
				switch c := container.(type) {
				case map[any]any:
					c[l.Sel.Name] = values[idx]
				case map[string]any:
					c[l.Sel.Name] = values[idx]
				default:
					// 使用反射缓存处理结构体字段赋值
					if err := globalReflectCache.set(container, l.Sel.Name, values[idx]); err != nil {
						return nil, err
					}
					// item := globalReflectCache.analyze(container)
					// if fieldInfo, ok := item.fields[l.Sel.Name]; ok {
					// 	v := reflect.ValueOf(container)
					// 	var base unsafe.Pointer
					// 	if v.Kind() == reflect.Ptr {
					// 		base = unsafe.Pointer(v.Pointer())
					// 	} else {
					// 		// 如果不是指针，创建一个临时指针
					// 		ptr := reflect.New(v.Type())
					// 		ptr.Elem().Set(v)
					// 		base = unsafe.Pointer(ptr.Pointer())
					// 		// 注意：这种情况下修改不会影响原始值，因为我们修改的是副本
					// 		// 可能需要返回错误或警告
					// 		return nil, fmt.Errorf("无法修改非指针结构体的字段: %s", l.Sel.Name)
					// 	}

					// 	// 获取字段的指针
					// 	ptr := unsafe.Pointer(uintptr(base) + fieldInfo.offset)
					// 	field := reflect.NewAt(fieldInfo.typ, ptr).Elem()

					// 	if !field.CanSet() {
					// 		return nil, fmt.Errorf("结构体字段 %s 不可写入（可能是未导出字段）", l.Sel.Name)
					// 	}

					// 	// 尝试设置字段值
					// 	fieldValue := reflect.ValueOf(values[idx])
					// 	if fieldValue.Type().AssignableTo(field.Type()) {
					// 		field.Set(fieldValue)
					// 		return values, nil
					// 	}

					// 	return nil, fmt.Errorf("类型不匹配：无法将 %T 赋值给 %s", values[idx], field.Type())
					// }
					// return nil, fmt.Errorf("不支持的选择器赋值操作: %T 没有字段 %s", container, l.Sel.Name)
				}
			default:
				return nil, fmt.Errorf("不支持的赋值目标类型: %T", l)
			}
		}
	case token.ADD_ASSIGN: // +=
		for idx, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("非左值表达式")
			}
			// 获取当前值
			currentVal, err := i.evalIdent(ident)
			if err != nil {
				return nil, err
			}
			// 计算新值
			newVal, err := add(currentVal, values[idx])
			if err != nil {
				return nil, err
			}
			// 更新值
			currentScope := i.scope
			for currentScope != nil {
				if _, ok := currentScope.Load(ident.Name); ok {
					currentScope.Store(ident.Name, newVal)
					break
				}
				currentScope = currentScope.parent
			}
		}
	default:
		return nil, fmt.Errorf("不支持的赋值操作符: %s", assign.Tok)
	}

	return values, nil
}

// 处理函数调用
func (i *Interpreter) evalCallExpr(call *ast.CallExpr) (any, error) {
	// 先评估函数表达式
	fn, err := i.eval(call.Fun)
	if err != nil {
		return nil, err
	}

	// 评估所有参数
	args := make([]any, len(call.Args))
	for idx, argExpr := range call.Args {
		argVal, err := i.eval(argExpr)
		if err != nil {
			return nil, err
		}
		args[idx] = argVal
	}

	// 根据函数类型进行不同的处理
	switch fn := fn.(type) {
	case func(...any) (any, error):
		// 闭包函数
		return fn(args...)
	case reflect.Value:
		// 内置函数
		reflectArgs := make([]reflect.Value, len(args))
		fnType := fn.Type()
		for idx, arg := range args {
			if arg == nil {
				var paramType reflect.Type
				if fnType.IsVariadic() && idx >= fnType.NumIn()-1 {
					paramType = fnType.In(fnType.NumIn() - 1).Elem()
				} else {
					paramType = fnType.In(idx)
				}
				reflectArgs[idx] = reflect.Zero(paramType)
			} else {
				reflectArgs[idx] = reflect.ValueOf(arg)
			}
		}
		results := fn.Call(reflectArgs)
		if len(results) == 0 {
			return nil, nil
		}
		return results[0].Interface(), nil
	case *Function:
		// 用户定义的函数
		// 创建新的作用域
		prevScope := i.scope
		newScope := &Scope{
			parent: prevScope,
		}
		i.scope = newScope
		defer func() { i.scope = prevScope }()

		// 处理参数
		if len(args) != len(fn.params) {
			return nil, fmt.Errorf("参数数量不匹配")
		}

		// 绑定参数到新作用域
		for idx, param := range fn.params {
			newScope.Store(param.Names[0].Name, args[idx])
		}

		// 执行函数体
		return i.eval(fn.body)
	default:
		// 使用反射处理其他类型的函数
		fnValue := reflect.ValueOf(fn)
		if fnValue.Kind() != reflect.Func {
			fmt.Printf("warn: 不是可调用的函数: %T \n", fn)
			return nil, nil
			//return nil, fmt.Errorf("不是可调用的函数: %T", fn)
		}

		// 准备参数
		fnType := fnValue.Type()
		if fnType.IsVariadic() {
			// 处理可变参数函数
			if len(args) < fnType.NumIn()-1 {
				return nil, fmt.Errorf("参数数量不足: 至少需要 %d 个参数, 得到 %d 个", fnType.NumIn()-1, len(args))
			}
		} else if fnType.NumIn() != len(args) {
			return nil, fmt.Errorf("参数数量不匹配: 期望 %d, 得到 %d", fnType.NumIn(), len(args))
		}

		callArgs := make([]reflect.Value, len(args))
		for i := 0; i < len(args); i++ {
			arg := args[i]
			var paramType reflect.Type
			if fnType.IsVariadic() && i >= fnType.NumIn()-1 {
				// 对于可变参数部分，使用可变参数的类型
				paramType = fnType.In(fnType.NumIn() - 1).Elem()
			} else {
				paramType = fnType.In(i)
			}

			if arg == nil {
				callArgs[i] = reflect.Zero(paramType)
			} else {
				argValue := reflect.ValueOf(arg)
				// 如果需要类型转换且可以转换，则进行转换
				if argValue.Type().ConvertibleTo(paramType) {
					callArgs[i] = argValue.Convert(paramType)
				} else {
					callArgs[i] = argValue
				}
			}
		}

		// 调用函数
		results := fnValue.Call(callArgs)
		if len(results) == 0 {
			return nil, nil
		}
		return results[0].Interface(), nil

		// return nil, fmt.Errorf("不是可调用的函数: %T", fn)
	}
}

// 处理二元表达式
func (i *Interpreter) evalBinaryExpr(expr *ast.BinaryExpr) (any, error) {
	left, err := i.eval(expr.X)
	if err != nil {
		return nil, err
	}

	right, err := i.eval(expr.Y)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case token.ADD:
		return add(left, right)
	case token.SUB:
		return sub(left, right)
	case token.MUL:
		return mul(left, right)
	case token.QUO:
		return div(left, right)
	case token.LSS:
		return lessThan(left, right)
	case token.GTR:
		return greaterThan(left, right)
	case token.LEQ:
		less, err := lessThan(left, right)
		if err != nil {
			return nil, err
		}
		equal, err := equal(left, right)
		if err != nil {
			return nil, err
		}
		return less || equal, nil
	case token.GEQ:
		greater, err := greaterThan(left, right)
		if err != nil {
			return nil, err
		}
		equal, err := equal(left, right)
		if err != nil {
			return nil, err
		}
		return greater || equal, nil
	case token.EQL:
		return equal(left, right)
	case token.REM:
		return mod(left, right)
	case token.LAND:
		return and(left, right)
	case token.LOR:
		return or(left, right)
	case token.NEQ: // !=
		equal, err := equal(left, right)
		if err != nil {
			return nil, err
		}
		return !equal, nil
	default:
		return nil, fmt.Errorf("不支持的运算符: %s", expr.Op)
	}
}

// 类型转换辅助函数
func toBool(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	default:
		if reflect.TypeOf(v).Kind() == reflect.Map || reflect.TypeOf(v).Kind() == reflect.Slice {
			return reflect.ValueOf(v).Len() > 0
		}
		return true
	}
}

// 算术运算实现
func add(a, b any) (any, error) {
	if a == nil {
		a = ""
	}
	if b == nil {
		b = ""
	}
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a + b, nil
		case float64:
			return float64(a) + b, nil
		case string:
			return fmt.Sprintf("%d%s", a, b), nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a + float64(b), nil
		case float64:
			return a + b, nil
		case string:
			return fmt.Sprintf("%f%s", a, b), nil
		}
	case string:
		switch b := b.(type) {
		case int:
			return fmt.Sprintf("%s%d", a, b), nil
		case float64:
			return fmt.Sprintf("%s%f", a, b), nil
		case string:
			return a + b, nil
		}
	}
	return nil, fmt.Errorf("类型不匹配: %T + %T", a, b)
}

// 减法运算
func sub(a, b any) (any, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a - b, nil
		case float64:
			return float64(a) - b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a - float64(b), nil
		case float64:
			return a - b, nil
		}
	}
	return nil, fmt.Errorf("无效操作: %T - %T", a, b)
}

// 乘法运算
func mul(a, b any) (any, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a * b, nil
		case float64:
			return float64(a) * b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a * float64(b), nil
		case float64:
			return a * b, nil
		}
	}
	return nil, fmt.Errorf("无效操作: %T * %T", a, b)
}

// 除法运算
func div(a, b any) (any, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			if b == 0 {
				return nil, fmt.Errorf("除以零错误")
			}
			return a / b, nil
		case float64:
			if b == 0 {
				return nil, fmt.Errorf("除以零错误")
			}
			return float64(a) / b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			if b == 0 {
				return nil, fmt.Errorf("除以零错误")
			}
			return a / float64(b), nil
		case float64:
			if b == 0 {
				return nil, fmt.Errorf("除以零错误")
			}
			return a / b, nil
		}
	}
	return nil, fmt.Errorf("无效操作: %T / %T", a, b)
}

// 比较运算
func lessThan(a, b any) (bool, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a < b, nil
		case float64:
			return float64(a) < b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a < float64(b), nil
		case float64:
			return a < b, nil
		}
	case string:
		if bStr, ok := b.(string); ok {
			return a < bStr, nil
		}
	}
	return false, fmt.Errorf("类型不匹配比较: %T < %T", a, b)
}

func greaterThan(a, b any) (bool, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a > b, nil
		case float64:
			return float64(a) > b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a > float64(b), nil
		case float64:
			return a > b, nil
		}
	case string:
		if bStr, ok := b.(string); ok {
			return a > bStr, nil
		}
	}
	return false, fmt.Errorf("类型不匹配比较: %T > %T", a, b)
}

func equal(a, b any) (bool, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a == b, nil
		case float64:
			return float64(a) == b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a == float64(b), nil
		case float64:
			return a == b, nil
		}
	case string:
		if bStr, ok := b.(string); ok {
			return a == bStr, nil
		}
	case bool:
		if bBool, ok := b.(bool); ok {
			return a == bBool, nil
		}
	case nil:
		return b == nil, nil
	}
	return false, nil // 类型不同直接返回false
}

// 取模运算（需要单独处理）
func mod(a, b any) (any, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			if b == 0 {
				return nil, fmt.Errorf("取模运算除数为零")
			}
			return a % b, nil
		}
	}
	return nil, fmt.Errorf("无效操作: %T %% %T", a, b)
}

func and(a, b any) (any, error) {
	left := toBool(a)
	if !left {
		return false, nil
	}
	right := toBool(b)
	return right, nil
}

func or(a, b any) (any, error) {
	left := toBool(a)
	if left {
		return true, nil
	}
	right := toBool(b)
	return right, nil
}

// 处理return语句的函数
func (i *Interpreter) evalReturnStmt(ret *ast.ReturnStmt) (any, error) {
	if len(ret.Results) == 0 {
		return nil, nil
	}
	// 目前只处理单个返回值
	return i.eval(ret.Results[0])
}

// 处理自增自减语句
func (i *Interpreter) evalIncDecStmt(stmt *ast.IncDecStmt) (any, error) {
	// 获取操作数
	ident, ok := stmt.X.(*ast.Ident)
	if !ok {
		return nil, fmt.Errorf("自增自减操作只支持变量")
	}

	// 获取当前值
	currentVal, err := i.evalIdent(ident)
	if err != nil {
		return nil, err
	}

	// 根据操作符计算新值
	var newVal any
	switch stmt.Tok {
	case token.INC: // ++
		newVal, err = add(currentVal, 1)
	case token.DEC: // --
		newVal, err = sub(currentVal, 1)
	}
	if err != nil {
		return nil, err
	}

	// 更新变量值
	currentScope := i.scope
	for currentScope != nil {
		if _, ok := currentScope.Load(ident.Name); ok {
			currentScope.Store(ident.Name, newVal)
			break
		}
		currentScope = currentScope.parent
	}

	return newVal, nil
}

// 新的处理函数字面量的方法
func (i *Interpreter) evalFuncLit(fn *ast.FuncLit) (any, error) {
	// 创建一个Function对象来存储函数信息
	function := &Function{
		params: fn.Type.Params.List,
		body:   fn.Body,
	}

	// 返回一个闭包函数
	return func(args ...any) (any, error) {
		// 创建新的作用域
		prevScope := i.scope
		newScope := &Scope{
			parent: prevScope,
		}
		i.scope = newScope
		defer func() { i.scope = prevScope }()

		// 绑定参数
		if len(args) != len(function.params) {
			return nil, fmt.Errorf("参数数量不匹配")
		}

		for idx, param := range function.params {
			newScope.Store(param.Names[0].Name, args[idx])
		}

		// 执行函数体
		return i.eval(function.body)
	}, nil
}

// 处理复合字面量
func (i *Interpreter) evalCompositeLit(lit *ast.CompositeLit) (any, error) {
	switch t := lit.Type.(type) {
	case *ast.MapType:
		// 创建map
		m := make(map[string]any)

		// 处理每个键值对
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				return nil, fmt.Errorf("map字面量必须是键值对")
			}

			// 计算键
			key, err := i.eval(kv.Key)
			if err != nil {
				return nil, err
			}

			// 计算值
			val, err := i.eval(kv.Value)
			if err != nil {
				return nil, err
			}

			m[key.(string)] = val
		}
		return m, nil

	case *ast.ArrayType:
		// 创建slice
		var slice []any
		for _, elt := range lit.Elts {
			val, err := i.eval(elt)
			if err != nil {
				return nil, err
			}
			slice = append(slice, val)
		}
		return slice, nil

	default:
		return nil, fmt.Errorf("不支持的复合字面量类型: %T", t)
	}
}

// 处理键值表达式
func (i *Interpreter) evalKeyValueExpr(kv *ast.KeyValueExpr) (any, error) {
	key, err := i.eval(kv.Key)
	if err != nil {
		return nil, err
	}

	value, err := i.eval(kv.Value)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"key":   key,
		"value": value,
	}, nil
}

// 处理索引表达式
func (i *Interpreter) evalIndexExpr(expr *ast.IndexExpr) (any, error) {
	// 计算被索引的对象
	container, err := i.eval(expr.X)
	if err != nil {
		return nil, err
	}

	// 计算索引值
	index, err := i.eval(expr.Index)
	if err != nil {
		return nil, err
	}

	// 根据容器类型进行不同的处理
	switch c := container.(type) {
	case map[any]any:
		// 对于map，直接使用索引作为键
		return c[index], nil

	case map[string]any:
		// 如果是string类型的map，需要将索引转换为string
		if strKey, ok := index.(string); ok {
			return c[strKey], nil
		}
		return nil, fmt.Errorf("map键必须是字符串类型，得到: %T", index)

	case []any:
		// 对于slice，索引必须是整数
		switch idx := index.(type) {
		case int:
			if idx < 0 || idx >= len(c) {
				return nil, fmt.Errorf("索引越界: %d", idx)
			}
			return c[idx], nil
		default:
			return nil, fmt.Errorf("slice索引必须是整数，得到: %T", index)
		}

	case string:
		// 对于字符串，索引必须是整数
		switch idx := index.(type) {
		case int:
			if idx < 0 || idx >= len(c) {
				return nil, fmt.Errorf("索引越界: %d", idx)
			}
			return string(c[idx]), nil
		default:
			return nil, fmt.Errorf("字符串索引必须是整数，得到: %T", index)
		}

	default:
		// 反射
		switch reflect.TypeOf(container).Kind() {
		case reflect.Map:
			// 使用map返回
			mapValue := reflect.ValueOf(container)
			keyValue := reflect.ValueOf(index)
			if val := mapValue.MapIndex(keyValue); val.IsValid() {
				return val.Interface(), nil
			}
		case reflect.Slice, reflect.Array:
			// 使用slice返回
			sliceValue := reflect.ValueOf(container)
			if idx, ok := index.(int); ok {
				if idx < 0 || idx >= sliceValue.Len() {
					return nil, fmt.Errorf("索引越界: %d", idx)
				}
				return sliceValue.Index(idx).Interface(), nil
			}
		}
		return nil, fmt.Errorf("不支持的索引操作: %T", container)
	}
}

// 处理选择器表达式的方法
func (i *Interpreter) evalSelectorExpr(sel *ast.SelectorExpr) (any, error) {
	// 计算被选择的对象
	container, err := i.eval(sel.X)
	if err != nil {
		return nil, err
	}

	if container == nil {
		fmt.Printf("warn: 选择器表达式对象为undefined: %v.%s \n", sel.X, sel.Sel.Name)
		return nil, nil
	}

	// 获取选择器名称
	fieldName := sel.Sel.Name

	// 处理map类型
	switch c := container.(type) {
	case map[any]any:
		if val, ok := c[fieldName]; ok {
			return val, nil
		}
	case map[string]any:
		if val, ok := c[fieldName]; ok {
			return val, nil
		}
	case map[string]string:
		if val, ok := c[fieldName]; ok {
			return val, nil
		}
	default:
		// 处理结构体和指针类型
		if item, _ := globalReflectCache.get(container, fieldName); item != nil {
			return item, nil
		}

		// 如果是map类型
		if reflect.TypeOf(container).Kind() == reflect.Map {
			// 使用map返回
			mapValue := reflect.ValueOf(container)
			keyValue := reflect.ValueOf(fieldName)
			if val := mapValue.MapIndex(keyValue); val.IsValid() {
				return val.Interface(), nil
			}
		}
	}

	return nil, fmt.Errorf("无法访问字段 %s: 对象类型 %T 不支持或字段不存在", fieldName, container)
}

// 处理声明语句的方法
func (i *Interpreter) evalDeclStmt(stmt *ast.DeclStmt) (any, error) {
	switch decl := stmt.Decl.(type) {
	case *ast.GenDecl:
		switch decl.Tok {
		case token.VAR:
			for _, spec := range decl.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					// 处理类型
					var varType reflect.Type
					if valueSpec.Type != nil {
						// 解析类型表达式
						typeExpr := valueSpec.Type
						resolvedType, err := i.resolveType(typeExpr)
						if err != nil {
							return nil, err
						}
						varType = resolvedType
					}

					// 处理初始值
					var value any = nil
					if len(valueSpec.Values) > 0 {
						var err error
						value, err = i.eval(valueSpec.Values[0])
						if err != nil {
							return nil, err
						}
					} else if varType != nil {
						// 如果有类型但没有初始值，创建零值
						// 对于结构体类型，创建指针
						if varType.Kind() == reflect.Struct {
							// 创建指向结构体的指针
							value = reflect.New(varType).Interface()
						} else {
							// 其他类型使用零值
							value = reflect.New(varType).Elem().Interface()
						}
					}

					// 为每个变量名赋值
					for _, name := range valueSpec.Names {
						i.scope.Store(name.Name, value)
					}
				}
			}
			return nil, nil
		}
	}
	return nil, fmt.Errorf("不支持的声明类型: %T", stmt.Decl)
}

// 解析类型表达式
func (i *Interpreter) resolveType(expr ast.Expr) (reflect.Type, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		// 简单标识符，如 int, string 等
		switch t.Name {
		case "int":
			return reflect.TypeOf(0), nil
		case "string":
			return reflect.TypeOf(""), nil
		case "bool":
			return reflect.TypeOf(false), nil
		case "float64":
			return reflect.TypeOf(0.0), nil
		default:
			return nil, fmt.Errorf("未知类型: %s", t.Name)
		}
	case *ast.SelectorExpr:
		// 包限定类型，如 strings.Builder
		if x, ok := t.X.(*ast.Ident); ok {
			packageName := x.Name

			// 从作用域中查找包
			pkg, err := i.evalIdent(x)
			if err != nil {
				return nil, fmt.Errorf("未找到包: %s", packageName)
			}

			// 检查包是否是 map
			pkgMap, ok := pkg.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s 不是一个包", packageName)
			}

			// 查找类型
			if typeObj, ok := pkgMap[t.Sel.Name]; ok {
				if reflectType, ok := typeObj.(reflect.Type); ok {
					return reflectType, nil
				}
				return nil, fmt.Errorf("%s.%s 不是类型", packageName, t.Sel.Name)
			}
			return nil, fmt.Errorf("包 %s 中没有类型 %s", packageName, t.Sel.Name)
		}
		return nil, fmt.Errorf("无效的类型选择器: %T", t.X)
	case *ast.ArrayType:
		// 数组或切片类型
		elemType, err := i.resolveType(t.Elt)
		if err != nil {
			return nil, err
		}
		if t.Len == nil {
			// 切片类型
			return reflect.SliceOf(elemType), nil
		}
		// 数组类型
		lenExpr, err := i.eval(t.Len)
		if err != nil {
			return nil, err
		}
		length, ok := lenExpr.(int)
		if !ok {
			return nil, fmt.Errorf("数组长度必须是整数")
		}
		return reflect.ArrayOf(length, elemType), nil
	case *ast.MapType:
		// Map 类型
		keyType, err := i.resolveType(t.Key)
		if err != nil {
			return nil, err
		}
		valueType, err := i.resolveType(t.Value)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(keyType, valueType), nil
	default:
		return nil, fmt.Errorf("不支持的类型表达式: %T", expr)
	}
}

// 处理分支语句的方法
func (i *Interpreter) evalBranchStmt(stmt *ast.BranchStmt) (any, error) {
	switch stmt.Tok {
	case token.BREAK:
		return breakSentinel{}, nil
	case token.CONTINUE:
		return continueSentinel{}, nil
	default:
		return nil, fmt.Errorf("不支持的分支语句类型: %v", stmt.Tok)
	}
}

// 定义哨兵类型用于处理 break 和 continue
type breakSentinel struct{}
type continueSentinel struct{}

func (i *Interpreter) evalRangeStmt(node *ast.RangeStmt) (any, error) {
	// 获取要遍历的值
	val, err := i.eval(node.X)
	if err != nil {
		return nil, err
	}

	// 使用反射获取值
	rval := reflect.ValueOf(val)

	switch rval.Kind() {
	case reflect.Slice, reflect.Array:
		// 遍历切片或数组
		for n := 0; n < rval.Len(); n++ {
			if node.Key != nil {
				// 设置索引变量
				i.scope.Store(node.Key.(*ast.Ident).Name, n)
			}
			if node.Value != nil {
				// 设置值变量
				i.scope.Store(node.Value.(*ast.Ident).Name, rval.Index(n).Interface())
			}
			// 执行循环体
			result, err := i.eval(node.Body)
			if err != nil {
				return nil, err
			}
			// 处理 break 和 continue
			if _, ok := result.(breakSentinel); ok {
				return nil, nil
			}
			if _, ok := result.(continueSentinel); ok {
				continue
			}
		}

	case reflect.Map:
		// 遍历 map
		iter := rval.MapRange()
		for iter.Next() {
			if node.Key != nil {
				// 设置键变量
				i.scope.Store(node.Key.(*ast.Ident).Name, iter.Key().Interface())
			}
			if node.Value != nil {
				// 设置值变量
				i.scope.Store(node.Value.(*ast.Ident).Name, iter.Value().Interface())
			}
			// 执行循环体
			result, err := i.eval(node.Body)
			if err != nil {
				return nil, err
			}
			// 处理 break 和 continue
			if _, ok := result.(breakSentinel); ok {
				return nil, nil
			}
			if _, ok := result.(continueSentinel); ok {
				continue
			}
		}

	default:
		return nil, fmt.Errorf("range: cannot range over %v (type %T)", val, val)
	}

	return nil, nil
}

// 处理一元表达式
func (i *Interpreter) evalUnaryExpr(expr *ast.UnaryExpr) (any, error) {
	// 计算操作数
	operand, err := i.eval(expr.X)
	if err != nil {
		return nil, err
	}

	switch expr.Op {
	case token.NOT: // !
		return !toBool(operand), nil
	case token.SUB: // -
		switch v := operand.(type) {
		case int:
			return -v, nil
		case float64:
			return -v, nil
		default:
			return nil, fmt.Errorf("一元减号操作不支持类型: %T", operand)
		}
	case token.ADD: // +
		switch v := operand.(type) {
		case int:
			return v, nil
		case float64:
			return v, nil
		default:
			return nil, fmt.Errorf("一元加号操作不支持类型: %T", operand)
		}
	default:
		return nil, fmt.Errorf("不支持的一元操作符: %v", expr.Op)
	}
}

// 处理 switch 语句
func (i *Interpreter) evalSwitchStmt(stmt *ast.SwitchStmt) (any, error) {
	// 如果有初始化语句，先执行
	if stmt.Init != nil {
		_, err := i.eval(stmt.Init)
		if err != nil {
			return nil, err
		}
	}

	// 计算 switch 表达式的值
	var tag any
	var err error
	if stmt.Tag != nil {
		tag, err = i.eval(stmt.Tag)
		if err != nil {
			return nil, err
		}
	}

	// 遍历所有的 case
	for _, caseClause := range stmt.Body.List {
		clause := caseClause.(*ast.CaseClause)

		// default 子句
		if clause.List == nil {
			if len(stmt.Body.List) > 0 {
				return i.evalBlockStmt(&ast.BlockStmt{List: clause.Body})
			}
			continue
		}

		// 检查每个 case 表达式
		for _, expr := range clause.List {
			caseVal, err := i.eval(expr)
			if err != nil {
				return nil, err
			}

			// 比较 case 值和 switch 表达式的值
			equal, err := equal(tag, caseVal)
			if err != nil {
				return nil, err
			}

			// 如果匹配，执行对应的语句块
			if equal {
				return i.evalBlockStmt(&ast.BlockStmt{List: clause.Body})
			}
		}
	}

	return nil, nil
}

func main() {
	interp := NewInterpreter()

	// 注册内置函数
	interp.Set("print", func(s any) {
		fmt.Printf("%v \n", s)
	})
	interp.Set("doTest", func(s map[string]any) {
		fmt.Printf("doTest %v \n", s)
	})
	interp.Set("doTest2", func(s ...string) {
		fmt.Printf("doTest2 %v \n", s)
	})
	// interp.BindGlobalObject(map[string]any{
	// 	"x": 1,
	// 	"y": func(a string) {
	// 		fmt.Printf("fomat by y   %v \n", a)
	// 	},
	// })

	interp.SetGlobal(test{
		basetest: basetest{
			X: 222,
		},
		basetest2: basetest2{
			YYY: 333,
		},
		Y: func(s string) {
			fmt.Printf("fomat by y   %v \n", s)
		},
		TestMp: []map[string]string{
			{
				"x": "foo",
				"y": "bar",
			},
			{
				"x": "foo2",
				"y": "bar2",
			},
		},
	})

	// 执行复杂逻辑
	code := `
	var abc = []any{1, 2, 3, 4, 5}
	for i, v := range abc {
		print(i + " " + v)
	}
	print(len(abc))
	no()
	TestArgs("foo", "bar", "baz")
	doTest2("foo", "bar")
	print(G.X)
	G.X = 3
	print(G.X)
	mp := map[string]any{}
	if mp {
		print("mp is not nil")
	}

	mp["x"] = "foo"
	mp["y"] = "bar"
	if mp {
		print("second mp is not nil")
	}
	print("测试string string map")
	mp2 := map[string]string{}
	mp2.x = "foo"
	mp2.y = "bar"
	print(mp2.x)
	print(mp2.y)

	print("测试[]map[string]string")
	for _, mp := range TestMp {
		print(mp.x)
		print(mp.y)
	}
	print("end")

	doTest(map[string]string{
		"x": "foo",
		"y": "bar",
	})
	a = a + 1
	Foo()
	print(X)
	print(YYY)
	print(G.X)
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
	if i % 2 == 0 && 1 != 0 {
		sum += i * 2
		print(sum)
	} else {
		sum += i
		print(sum)
	}
}
	mp := map[string]any{
		"x": 1,
		"y": 2,
	}
	for k, v := range mp {
		print(k + " " + v)
	}
return sum
`
	result, err := interp.Interpret(code)
	if err != nil {
		panic(err)
	}
	fmt.Println("计算结果:", result) // 应该输出 1 + 4 + 3 + 8 + 5 = 21
}
