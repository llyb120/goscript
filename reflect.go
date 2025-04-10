package goscript

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// 对反射进行加速

type reflectCache struct {
	sync.RWMutex
	cache map[reflect.Type]*reflectCacheItem
}

type fieldInfo struct {
	index []int // 字段的索引路径
	typ   reflect.Type
}

type methodInfo struct {
	method  reflect.Method
	pointer bool // true表示是指针接收器的方法
	offset  int  // 方法在接口表中的偏移量
}

type reflectCacheItem struct {
	fields        map[string]fieldInfo
	methods       map[string]methodInfo
	embeddedTypes []reflect.Type
}

func NewReflectCache() *reflectCache {
	return &reflectCache{
		cache: make(map[reflect.Type]*reflectCacheItem),
	}
}

func (r *reflectCache) analyze(val any) *reflectCacheItem {
	t := reflect.TypeOf(val)

	// 先检查是否是接口类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Interface {
		// 获取接口中实际的值的类型
		iface := (*[2]unsafe.Pointer)(unsafe.Pointer(&val))
		if iface[1] != nil {
			t = reflect.TypeOf(*(*interface{})(iface[1]))
		}
	}

	// 尝试读取缓存
	r.RLock()
	if item, ok := r.cache[t]; ok {
		r.RUnlock()
		return item
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()

	// double check
	if item, ok := r.cache[t]; ok {
		return item
	}

	item := r.getOrCreateCacheItem(t)
	r.cache[t] = item
	return item
}

// 获取或创建缓存项
func (r *reflectCache) getOrCreateCacheItem(t reflect.Type) *reflectCacheItem {
	// 检查缓存中是否已存在
	if item, ok := r.cache[t]; ok {
		return item
	}

	item := &reflectCacheItem{
		fields:        make(map[string]fieldInfo),
		methods:       make(map[string]methodInfo),
		embeddedTypes: []reflect.Type{},
	}

	// 处理结构体
	if t.Kind() == reflect.Struct {
		// 缓存字段和嵌入类型
		r.cacheFields(item, t)
	}

	// 缓存方法（包括值接收器和指针接收器的方法）
	r.cacheMethods(item, t)

	return item
}

// 缓存字段
func (r *reflectCache) cacheFields(item *reflectCacheItem, t reflect.Type) {
	var cacheField func(reflect.Type, []int)

	cacheField = func(t reflect.Type, parentIndex []int) {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			index := append(append([]int{}, parentIndex...), i)

			if field.IsExported() || field.Anonymous {
				item.fields[field.Name] = fieldInfo{
					index: index,
					typ:   field.Type,
				}

				// 处理嵌入字段
				if field.Anonymous {
					fieldType := field.Type
					if fieldType.Kind() == reflect.Ptr {
						fieldType = fieldType.Elem()
					}
					if fieldType.Kind() == reflect.Struct {
						item.embeddedTypes = append(item.embeddedTypes, fieldType)

						// 递归处理嵌入字段
						embeddedItem := r.getOrCreateCacheItem(fieldType)

						// 缓存嵌入字段的字段
						for name, info := range embeddedItem.fields {
							if _, exists := item.fields[name]; !exists {
								// 计算完整的索引路径
								fullIndex := append(append([]int{}, index...), info.index...)
								item.fields[name] = fieldInfo{
									index: fullIndex,
									typ:   info.typ,
								}
							}
						}

						// 缓存嵌入字段的方法
						for methodName, methodInfo := range embeddedItem.methods {
							if _, exists := item.methods[methodName]; !exists {
								item.methods[methodName] = methodInfo
							}
						}

						// 递归处理嵌入字段的结构
						cacheField(fieldType, index)
					}
				}
			}
		}
	}

	cacheField(t, nil)
}

// 缓存方法
func (r *reflectCache) cacheMethods(item *reflectCacheItem, t reflect.Type) {
	// 先获取嵌入结构体的方法
	for _, embedded := range item.embeddedTypes {
		// 获取嵌入类型的缓存项
		embeddedItem := r.getOrCreateCacheItem(embedded)

		// 缓存嵌入类型的方法
		for methodName, methodInfo := range embeddedItem.methods {
			// 只有当方法不存在时才添加嵌入类型的方法
			if _, exists := item.methods[methodName]; !exists {
				item.methods[methodName] = methodInfo
			}
		}
	}

	// 缓存值接收器的方法（这些方法会覆盖嵌入类型的方法）
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.IsExported() {
			item.methods[method.Name] = methodInfo{
				method:  method,
				pointer: false,
				offset:  i,
			}
		}
	}

	// 缓存指针接收器的方法（这些方法会覆盖值接收器和嵌入类型的方法）
	ptrType := reflect.PtrTo(t)
	for i := 0; i < ptrType.NumMethod(); i++ {
		method := ptrType.Method(i)
		if method.IsExported() {
			item.methods[method.Name] = methodInfo{
				method:  method,
				pointer: true,
				offset:  i,
			}
		}
	}
}

// 获取字段值
func (r *reflectCache) getValue(item *reflectCacheItem, obj any, fieldName string) (any, reflect.Type) {
	if item == nil {
		item = r.analyze(obj)
	}
	if field, ok := item.fields[fieldName]; ok {
		v := reflect.ValueOf(obj)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		return v.FieldByIndex(field.index).Interface(), field.typ
	}
	return nil, nil
}

// 获取方法
func (r *reflectCache) getMethod(item *reflectCacheItem, obj any, methodName string) (any, reflect.Type) {
	if item == nil {
		item = r.analyze(obj)
	}

	if method, ok := item.methods[methodName]; ok {
		v := reflect.ValueOf(obj)

		// 处理接收器类型不匹配的情况
		if method.pointer {
			// 需要指针接收器
			if v.Kind() != reflect.Ptr {
				// 如果当前值不是指针，创建新的指针
				newPtr := reflect.New(v.Type())
				newPtr.Elem().Set(v)
				v = newPtr
			}
		} else {
			// 需要值接收器
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
		}

		// 使用偏移量获取方法
		m := v.Method(method.offset)
		return m.Interface(), method.method.Type
	}
	return nil, nil
}

func (r *reflectCache) get(obj any, name string) (any, reflect.Type) {
	item := r.analyze(obj)
	if item == nil {
		return nil, nil
	}
	if field, typ := r.getValue(item, obj, name); field != nil {
		return field, typ
	}
	if method, typ := r.getMethod(item, obj, name); method != nil {
		return method, typ
	}
	return nil, nil
}

// set 设置对象的字段值，支持指针类型
func (r *reflectCache) set(obj any, fieldName string, value any) error {
	item := r.analyze(obj)
	if item == nil {
		return fmt.Errorf("failed to analyze object")
	}

	field, ok := item.fields[fieldName]
	if !ok {
		return fmt.Errorf("field %s not found", fieldName)
	}

	valueToSet := reflect.ValueOf(value)
	if !valueToSet.Type().AssignableTo(field.typ) {
		return fmt.Errorf("cannot assign value of type %s to field of type %s", valueToSet.Type(), field.typ)
	}

	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot set field on non-pointer value")
	}

	// 处理 *interface{} 的情况
	if v.Type().Elem().Kind() == reflect.Interface {
		elem := v.Elem()
		if elem.IsNil() {
			return fmt.Errorf("interface value is nil")
		}
		// 获取接口中的实际值
		actualValue := elem.Elem()

		// 创建可设置的副本
		copyValue := reflect.New(actualValue.Type()).Elem()
		copyValue.Set(actualValue)

		// 获取字段值
		fieldValue := copyValue.FieldByIndex(field.index)
		if !fieldValue.CanSet() {
			// 尝试通过指针访问
			if copyValue.CanAddr() {
				fieldValue = reflect.NewAt(copyValue.Type(), unsafe.Pointer(copyValue.UnsafeAddr())).Elem().FieldByIndex(field.index)
			}
		}

		if !fieldValue.CanSet() {
			return fmt.Errorf("cannot set field %s: field is not settable", fieldName)
		}

		// 类型转换处理
		convertedValue, err := convertType(reflect.ValueOf(value), fieldValue.Type())
		if err != nil {
			return err
		}

		fieldValue.Set(convertedValue)
		elem.Set(copyValue)
		return nil
	} else {
		v = v.Elem()
	}
	fieldValue := v.FieldByIndex(field.index)
	if !fieldValue.CanSet() {
		return fmt.Errorf("cannot set field %s: field is not settable", fieldName)
	}

	fieldValue.Set(valueToSet)
	return nil
}

// 新增类型转换函数
func convertType(src reflect.Value, dstType reflect.Type) (reflect.Value, error) {
	if src.Type().ConvertibleTo(dstType) {
		return src.Convert(dstType), nil
	}
	if src.Kind() == reflect.Interface {
		src = src.Elem()
	}
	if src.Type().ConvertibleTo(dstType) {
		return src.Convert(dstType), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot convert %s to %s", src.Type(), dstType)
}
