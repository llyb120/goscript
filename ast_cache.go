package goscript

import (
	"go/ast"
	"sync"
)

type astCache struct {
	sync.RWMutex
	cache map[string]*ast.File
}

func (c *astCache) GetIfNotExist(key string, fn func() (*ast.File, error)) (*ast.File, error) {
	res := (func() *ast.File {
		c.RLock()
		defer c.RUnlock()
		if value, ok := c.cache[key]; ok {
			return value
		}
		return nil
	})()
	if res != nil {
		return res, nil
	}
	c.Lock()
	defer c.Unlock()
	// double check
	if value, ok := c.cache[key]; ok {
		return value, nil
	}
	res, err := fn()
	if err != nil {
		return nil, err
	}
	c.cache[key] = res
	return res, nil
}
