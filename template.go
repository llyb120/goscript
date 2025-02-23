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

type Function struct {
	params []*ast.Field
	body   *ast.BlockStmt
}

type Interpreter struct {
	scope     *Scope
	funcTable map[string]reflect.Value // 内置函数表
	userFuncs map[string]*Function     // 用户定义的函数表
}

type Scope struct {
	parent  *Scope
	objects map[string]interface{}
}

func NewInterpreter() *Interpreter {
	globalScope := &Scope{
		objects: make(map[string]interface{}),
	}
	return &Interpreter{
		scope:     globalScope,
		funcTable: make(map[string]reflect.Value),
		userFuncs: make(map[string]*Function),
	}
}

func (i *Interpreter) RegisterFunction(name string, fn interface{}) {
	i.funcTable[name] = reflect.ValueOf(fn)
}

func (i *Interpreter) Interpret(code string) (interface{}, error) {
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

func (i *Interpreter) eval(node ast.Node) (interface{}, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return i.evalBasicLit(n)
	case *ast.Ident:
		return i.evalIdent(n)
	case *ast.BinaryExpr:
		return i.evalBinaryExpr(n)
	case *ast.CallExpr:
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
	default:
		return nil, fmt.Errorf("unsupported node type: %T", node)
	}
}

// 基础类型处理
func (i *Interpreter) evalBasicLit(lit *ast.BasicLit) (interface{}, error) {
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
func (i *Interpreter) evalIdent(ident *ast.Ident) (interface{}, error) {
	// 先检查是否是预定义常量
	switch ident.Name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil
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

	return nil, fmt.Errorf("未定义的标识符: %s", ident.Name)
}

// 处理代码块
func (i *Interpreter) evalBlockStmt(block *ast.BlockStmt) (interface{}, error) {
	// 创建新的作用域
	prevScope := i.scope
	i.scope = &Scope{
		parent:  prevScope,
		objects: make(map[string]interface{}),
	}
	defer func() { i.scope = prevScope }() // 确保作用域恢复

	var result interface{}
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
func (i *Interpreter) evalForStmt(f *ast.ForStmt) (interface{}, error) {
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
		_, err := i.eval(f.Body)
		if err != nil {
			return nil, err
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
func (i *Interpreter) evalIfStmt(ifStmt *ast.IfStmt) (interface{}, error) {
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
func (i *Interpreter) evalAssignStmt(assign *ast.AssignStmt) (interface{}, error) {
	// 处理右侧表达式
	values := make([]interface{}, len(assign.Rhs))
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
				case map[interface{}]interface{}:
					c[index] = values[idx]
				case map[string]interface{}:
					if strKey, ok := index.(string); ok {
						c[strKey] = values[idx]
					} else {
						return nil, fmt.Errorf("map键必须是字符串类型")
					}
				case []interface{}:
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
func (i *Interpreter) evalCallExpr(call *ast.CallExpr) (interface{}, error) {
	// 先评估函数表达式
	fn, err := i.eval(call.Fun)
	if err != nil {
		return nil, err
	}

	// 评估所有参数
	args := make([]interface{}, len(call.Args))
	for idx, argExpr := range call.Args {
		argVal, err := i.eval(argExpr)
		if err != nil {
			return nil, err
		}
		args[idx] = argVal
	}

	// 根据函数类型进行不同的处理
	switch fn := fn.(type) {
	case func(...interface{}) (interface{}, error):
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
			objects: make(map[string]interface{}),
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
		return nil, fmt.Errorf("不是可调用的函数: %T", fn)
	}
}

// 处理二元表达式
func (i *Interpreter) evalBinaryExpr(expr *ast.BinaryExpr) (interface{}, error) {
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
func toBool(val interface{}) bool {
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
func add(a, b interface{}) (interface{}, error) {
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
func sub(a, b interface{}) (interface{}, error) {
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
func mul(a, b interface{}) (interface{}, error) {
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
func div(a, b interface{}) (interface{}, error) {
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
func lessThan(a, b interface{}) (bool, error) {
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

func greaterThan(a, b interface{}) (bool, error) {
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

func equal(a, b interface{}) (bool, error) {
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
func mod(a, b interface{}) (interface{}, error) {
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

// 添加处理return语句的函数
func (i *Interpreter) evalReturnStmt(ret *ast.ReturnStmt) (interface{}, error) {
	if len(ret.Results) == 0 {
		return nil, nil
	}
	// 目前只处理单个返回值
	return i.eval(ret.Results[0])
}

// 处理自增自减语句
func (i *Interpreter) evalIncDecStmt(stmt *ast.IncDecStmt) (interface{}, error) {
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
	var newVal interface{}
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

// 添加新的处理函数字面量的方法
func (i *Interpreter) evalFuncLit(fn *ast.FuncLit) (interface{}, error) {
	// 创建一个Function对象来存储函数信息
	function := &Function{
		params: fn.Type.Params.List,
		body:   fn.Body,
	}

	// 返回一个闭包函数
	return func(args ...interface{}) (interface{}, error) {
		// 创建新的作用域
		prevScope := i.scope
		newScope := &Scope{
			parent:  prevScope,
			objects: make(map[string]interface{}),
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
func (i *Interpreter) evalCompositeLit(lit *ast.CompositeLit) (interface{}, error) {
	switch t := lit.Type.(type) {
	case *ast.MapType:
		// 创建map
		m := make(map[interface{}]interface{})

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
		var slice []interface{}
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
func (i *Interpreter) evalKeyValueExpr(kv *ast.KeyValueExpr) (interface{}, error) {
	key, err := i.eval(kv.Key)
	if err != nil {
		return nil, err
	}

	value, err := i.eval(kv.Value)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"key":   key,
		"value": value,
	}, nil
}

// 处理索引表达式
func (i *Interpreter) evalIndexExpr(expr *ast.IndexExpr) (interface{}, error) {
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
	case map[interface{}]interface{}:
		// 对于map，直接使用索引作为键
		return c[index], nil

	case map[string]interface{}:
		// 如果是string类型的map，需要将索引转换为string
		if strKey, ok := index.(string); ok {
			return c[strKey], nil
		}
		return nil, fmt.Errorf("map键必须是字符串类型，得到: %T", index)

	case []interface{}:
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

// 添加处理选择器表达式的方法
func (i *Interpreter) evalSelectorExpr(sel *ast.SelectorExpr) (interface{}, error) {
	// 计算被选择的对象
	container, err := i.eval(sel.X)
	if err != nil {
		return nil, err
	}

	// 获取选择器名称
	fieldName := sel.Sel.Name

	// 处理map类型
	switch c := container.(type) {
	case map[interface{}]interface{}:
		if val, ok := c[fieldName]; ok {
			return val, nil
		}
	case map[string]interface{}:
		if val, ok := c[fieldName]; ok {
			return val, nil
		}
	}

	return nil, fmt.Errorf("无法访问字段 %s: 对象类型 %T 不支持或字段不存在", fieldName, container)
}

func main() {
	interp := NewInterpreter()

	// 注册内置函数
	interp.RegisterFunction("print", func(s any) {
		fmt.Printf("%v \n", s)
	})
	interp.scope.objects["x"] = 1

	// 执行复杂逻辑
	code := `
	yyy := func(a string){
		print("ok " + a)
	}
	yyy("shit")
	mp := map[string]any{
		"x": 1,
		"y": 2,
	}
	mp["x"] = "shit"
	mp["y"] = 4
	mp.x = "shit"

	yyy(mp.x)
	
sum := 0
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
`

	result, err := interp.Interpret(code)
	if err != nil {
		panic(err)
	}
	fmt.Println("计算结果:", result) // 应该输出 1 + 4 + 3 + 8 + 5 = 21
}
