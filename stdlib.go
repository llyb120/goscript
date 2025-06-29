package goscript

import (
	"fmt"
	"reflect"
	"strings"
)

// 注册标准库包
func (i *Interpreter) libs() {

	// strings 包
	i.Set("strings", map[string]any{
		"Builder":        reflect.TypeOf(strings.Builder{}),
		"Join":           strings.Join,
		"Split":          strings.Split,
		"Compare":        strings.Compare,
		"Contains":       strings.Contains,
		"ContainsAny":    strings.ContainsAny,
		"ContainsRune":   strings.ContainsRune,
		"Count":          strings.Count,
		"EqualFold":      strings.EqualFold,
		"Fields":         strings.Fields,
		"FieldsFunc":     strings.FieldsFunc,
		"HasPrefix":      strings.HasPrefix,
		"HasSuffix":      strings.HasSuffix,
		"Index":          strings.Index,
		"IndexAny":       strings.IndexAny,
		"IndexByte":      strings.IndexByte,
		"IndexFunc":      strings.IndexFunc,
		"IndexRune":      strings.IndexRune,
		"LastIndex":      strings.LastIndex,
		"LastIndexAny":   strings.LastIndexAny,
		"LastIndexByte":  strings.LastIndexByte,
		"LastIndexFunc":  strings.LastIndexFunc,
		"Map":            strings.Map,
		"Repeat":         strings.Repeat,
		"Replace":        strings.Replace,
		"ReplaceAll":     strings.ReplaceAll,
		"SplitAfter":     strings.SplitAfter,
		"SplitAfterN":    strings.SplitAfterN,
		"SplitN":         strings.SplitN,
		"Title":          strings.Title,
		"ToLower":        strings.ToLower,
		"ToLowerSpecial": strings.ToLowerSpecial,
		"ToTitle":        strings.ToTitle,
		"ToTitleSpecial": strings.ToTitleSpecial,
		"ToUpper":        strings.ToUpper,
		"ToUpperSpecial": strings.ToUpperSpecial,
		"Trim":           strings.Trim,
		"TrimFunc":       strings.TrimFunc,
		"TrimLeft":       strings.TrimLeft,
		"TrimLeftFunc":   strings.TrimLeftFunc,
		"TrimPrefix":     strings.TrimPrefix,
		"TrimRight":      strings.TrimRight,
		"TrimRightFunc":  strings.TrimRightFunc,
		"TrimSpace":      strings.TrimSpace,
		"TrimSuffix":     strings.TrimSuffix,
	})

	// fmt 包
	i.Set("fmt", map[string]any{
		"Println": fmt.Println,
		"Printf":  fmt.Printf,
		"Sprintf": fmt.Sprintf,
	})

	i.Set("len", func(v any) int {
		return reflect.ValueOf(v).Len()
	})

	i.Set("print", func(args ...any) {
		fmt.Println(args...)
	})

	// has 函数：检查容器是否包含指定的元素或字段
	// 对于数组和slice：检查是否包含指定的值
	// 对于map：检查是否包含指定的key
	// 对于结构体：使用reflect缓存检查是否包含指定的字段
	i.Set("has", func(container any, element ...any) bool {
		if container == nil {
			return false
		}

		var flag = true
		containerVal := reflect.ValueOf(container)
		containerType := containerVal.Type()

		// 处理指针类型
		for containerType.Kind() == reflect.Ptr || containerType.Kind() == reflect.Interface {
			if containerVal.IsNil() {
				return false
			}
			containerVal = containerVal.Elem()
			containerType = containerVal.Type()
		}

		for _, elem := range element {
			elem := elem
			flag = flag && func() bool {
				switch containerType.Kind() {
				case reflect.Slice, reflect.Array:
					// 对于数组和slice，检查是否包含指定的值
					for i := 0; i < containerVal.Len(); i++ {
						item := containerVal.Index(i)
						if reflect.DeepEqual(item.Interface(), elem) {
							return true
						}
					}
					return false

				case reflect.Map:
					// 对于map，检查是否包含指定的key
					keyVal := reflect.ValueOf(elem)
					mapKeyType := containerVal.Type().Key()

					// 检查key类型是否兼容
					if !keyVal.Type().AssignableTo(mapKeyType) {
						// 尝试类型转换
						if keyVal.Type().ConvertibleTo(mapKeyType) {
							keyVal = keyVal.Convert(mapKeyType)
						} else {
							return false
						}
					}

					// 检查key是否存在
					value := containerVal.MapIndex(keyVal)
					return value.IsValid()

				case reflect.Struct:
					// 对于结构体，使用reflect缓存检查是否包含指定的字段
					fieldNameStr, ok := elem.(string)
					if !ok {
						return false
					}

					// 使用全局反射缓存
					cacheItem := globalReflectCache.analyze(container)

					// 检查字段是否存在
					_, exists := cacheItem.fields[fieldNameStr]
					return exists

				case reflect.String:
					// 对于字符串，检查是否包含指定的子字符串
					strVal, ok := elem.(string)
					if !ok {
						return false
					}
					return strings.Contains(containerVal.String(), strVal)
				default:
					return false
				}
			}()
		}

		return flag
	})

}
