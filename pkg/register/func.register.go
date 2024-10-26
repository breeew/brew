package register

import "sync"

type funcRegister struct {
	handlers map[any][]Handler
	locker   sync.Mutex
}

var fr *funcRegister

func init() {
	fr = &funcRegister{
		handlers: make(map[any][]Handler),
	}
}

type Handler func()

func RegisterFunc(key any, handler Handler) {
	fr.locker.Lock()
	fr.handlers[key] = append(fr.handlers[key], handler)
	fr.locker.Unlock()
}

func ResolveFuncHandlers(key any) []Handler {
	fr.locker.Lock()
	defer fr.locker.Unlock()
	return fr.handlers[key]
}
