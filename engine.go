package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
)

var globalReflectCache = NewReflectCache()

type Function struct {
	params []*ast.Field
	body   *ast.BlockStmt
}

type Interpreter struct {
	scope     *Scope
	global    any
	funcTable map[string]reflect.Value // 内置函数表
	userFuncs map[string]*Function     // 用户定义的函数表
}

type Scope struct {
	parent  *Scope
	objects map[string]any
}

func NewInterpreter() *Interpreter {
	globalScope := &Scope{
		objects: make(map[string]any),
	}
	return &Interpreter{
		scope:     globalScope,
		global:    globalScope,
		funcTable: make(map[string]reflect.Value),
		userFuncs: make(map[string]*Function),
	}
}

func (i *Interpreter) RegisterFunction(name string, fn any) {
	i.funcTable[name] = reflect.ValueOf(fn)
}

func (i *Interpreter) BindObject(name string, obj any) {
	i.scope.objects[name] = obj
}

func (i *Interpreter) BindGlobalObject(obj any) {
	i.global = obj
}

func (i *Interpreter) Interpret(code string) (any, error) {
	fset := token.NewFileSet()
	code = `package main
	func __main__() any {	
	` + code + `
	}
	`
	fmt.Println(code)
	f, err := parser.ParseFile(fset, "", code, parser.Mode(0))
	if err != nil {
		return nil, err
	}

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

	return i.eval(f.Decls[0].(*ast.FuncDecl).Body)
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
		if val, ok := currentScope.objects[ident.Name]; ok {
			return val, nil
		}
		currentScope = currentScope.parent
	}

	// 检查是否是内置函数
	if fn, ok := i.funcTable[ident.Name]; ok {
		return fn, nil
	}

	// 检查是否是用户定义的函数
	if fn, ok := i.userFuncs[ident.Name]; ok {
		return fn, nil
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

	return nil, fmt.Errorf("未定义的标识符: %s", ident.Name)
}

// 处理代码块
func (i *Interpreter) evalBlockStmt(block *ast.BlockStmt) (any, error) {
	// 创建新的作用域
	prevScope := i.scope
	i.scope = &Scope{
		parent:  prevScope,
		objects: make(map[string]any),
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
					i.scope.objects[l.Name] = values[idx]
				} else {
					// 查找变量并赋值
					currentScope := i.scope
					for currentScope != nil {
						if _, ok := currentScope.objects[l.Name]; ok {
							currentScope.objects[l.Name] = values[idx]
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
					return nil, fmt.Errorf("不支持的选择器赋值操作: %T", container)
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
				if _, ok := currentScope.objects[ident.Name]; ok {
					currentScope.objects[ident.Name] = newVal
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
		for idx, arg := range args {
			reflectArgs[idx] = reflect.ValueOf(arg)
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
			parent:  prevScope,
			objects: make(map[string]any),
		}
		i.scope = newScope
		defer func() { i.scope = prevScope }()

		// 处理参数
		if len(args) != len(fn.params) {
			return nil, fmt.Errorf("参数数量不匹配")
		}

		// 绑定参数到新作用域
		for idx, param := range fn.params {
			newScope.objects[param.Names[0].Name] = args[idx]
		}

		// 执行函数体
		return i.eval(fn.body)
	default:
		// 使用反射处理其他类型的函数
		fnValue := reflect.ValueOf(fn)
		if fnValue.Kind() != reflect.Func {
			return nil, fmt.Errorf("不是可调用的函数: %T", fn)
		}

		// 准备参数
		fnType := fnValue.Type()
		if fnType.NumIn() != len(args) {
			return nil, fmt.Errorf("参数数量不匹配: 期望 %d, 得到 %d", fnType.NumIn(), len(args))
		}

		callArgs := make([]reflect.Value, len(args))
		for i := 0; i < len(args); i++ {
			arg := args[i]
			if arg == nil {
				callArgs[i] = reflect.Zero(fnType.In(i))
			} else {
				callArgs[i] = reflect.ValueOf(arg)
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
		return true
	}
}

// 算术运算实现
func add(a, b any) (any, error) {
	switch a := a.(type) {
	case int:
		switch b := b.(type) {
		case int:
			return a + b, nil
		case float64:
			return float64(a) + b, nil
		}
	case float64:
		switch b := b.(type) {
		case int:
			return a + float64(b), nil
		case float64:
			return a + b, nil
		}
	case string:
		if bStr, ok := b.(string); ok {
			return a + bStr, nil
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
		if _, ok := currentScope.objects[ident.Name]; ok {
			currentScope.objects[ident.Name] = newVal
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
			parent:  prevScope,
			objects: make(map[string]any),
		}
		i.scope = newScope
		defer func() { i.scope = prevScope }()

		// 绑定参数
		if len(args) != len(function.params) {
			return nil, fmt.Errorf("参数数量不匹配")
		}

		for idx, param := range function.params {
			newScope.objects[param.Names[0].Name] = args[idx]
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
		m := make(map[any]any)

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

			m[key] = val
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
	default:
		// 处理结构体和指针类型
		if item, _ := globalReflectCache.get(container, fieldName); item != nil {
			return item, nil
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
					// 处理初始值
					var value any = nil
					if len(valueSpec.Values) > 0 {
						var err error
						value, err = i.eval(valueSpec.Values[0])
						if err != nil {
							return nil, err
						}
					}
					// 为每个变量名赋值
					for _, name := range valueSpec.Names {
						i.scope.objects[name.Name] = value
					}
				}
			}
			return nil, nil
		}
	}
	return nil, fmt.Errorf("不支持的声明类型: %T", stmt.Decl)
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

func main() {
	interp := NewInterpreter()

	// 注册内置函数
	interp.RegisterFunction("print", func(s any) {
		fmt.Printf("%v \n", s)
	})
	// interp.BindGlobalObject(map[string]any{
	// 	"x": 1,
	// 	"y": func(a string) {
	// 		fmt.Printf("fomat by y   %v \n", a)
	// 	},
	// })

	interp.BindGlobalObject(test{
		X: 222,
		Y: func(s string) {
			fmt.Printf("fomat by y   %v \n", s)
		},
	})

	// 执行复杂逻辑
	code := `
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
`

	result, err := interp.Interpret(code)
	if err != nil {
		panic(err)
	}
	fmt.Println("计算结果:", result) // 应该输出 1 + 4 + 3 + 8 + 5 = 21
}
