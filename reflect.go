package main

import (
	"reflect"
	"sync"
	"unsafe"
)

// 对反射进行加速

type reflectCache struct {
	sync.RWMutex
	cache map[reflect.Type]reflectCacheItem
}

type fieldInfo struct {
	offset uintptr
	typ    reflect.Type
}

type methodInfo struct {
	method  reflect.Method
	pointer bool // true表示是指针接收器的方法
}

type reflectCacheItem struct {
	fields  map[string]fieldInfo
	methods map[string]methodInfo
}

func NewReflectCache() *reflectCache {
	return &reflectCache{
		cache: make(map[reflect.Type]reflectCacheItem),
	}
}

func (r *reflectCache) get(val any) reflectCacheItem {
	t := reflect.TypeOf(val)
	// originalType := t
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
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

	item := reflectCacheItem{
		fields:  make(map[string]fieldInfo),
		methods: make(map[string]methodInfo),
	}

	// 缓存字段
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.IsExported() {
				item.fields[field.Name] = fieldInfo{
					offset: field.Offset,
					typ:    field.Type,
				}
			}
		}

		// 缓存值接收器的方法
		for i := 0; i < t.NumMethod(); i++ {
			method := t.Method(i)
			if method.IsExported() {
				item.methods[method.Name] = methodInfo{
					method:  method,
					pointer: false,
				}
			}
		}

		// 缓存指针接收器的方法
		ptrType := reflect.PtrTo(t)
		for i := 0; i < ptrType.NumMethod(); i++ {
			method := ptrType.Method(i)
			if method.IsExported() {
				// 如果值方法中没有这个方法，或者这就是一个指针方法
				if _, exists := item.methods[method.Name]; !exists {
					item.methods[method.Name] = methodInfo{
						method:  method,
						pointer: true,
					}
				}
			}
		}
	}

	r.cache[t] = item
	return item
}

// 获取字段值
func (r *reflectCache) getValue(obj any, fieldName string) (any, reflect.Type) {
	item := r.get(obj)
	if field, ok := item.fields[fieldName]; ok {
		v := reflect.ValueOf(obj)
		var base unsafe.Pointer
		if v.Kind() == reflect.Ptr {
			base = unsafe.Pointer(v.Pointer())
		} else {
			// 值类型需要先获取数据的地址
			ptr := reflect.New(v.Type())
			ptr.Elem().Set(v)
			base = unsafe.Pointer(ptr.Pointer())
		}

		// 统一使用反射获取字段值
		ptr := unsafe.Pointer(uintptr(base) + field.offset)
		return reflect.NewAt(field.typ, ptr).Elem().Interface(), field.typ
	}
	return nil, nil
}

// 获取方法
func (r *reflectCache) getMethod(obj any, methodName string) (reflect.Method, bool, bool) {
	item := r.get(obj)
	if method, ok := item.methods[methodName]; ok {
		return method.method, method.pointer, true
	}
	return reflect.Method{}, false, false
}
