package goscript

import "sync"

type Scope struct {
	sync.Map
	parent *Scope
}

type SharedScope struct {
	sync.Map
}
