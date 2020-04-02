package wasmtime

// #include <wasmtime.h>
// #include "shims.h"
import "C"
import "runtime"
import "errors"

type Linker struct {
	_ptr  *C.wasmtime_linker_t
	Store *Store
}

func NewLinker(store *Store) *Linker {
	ptr := C.wasmtime_linker_new(store.ptr())
	runtime.KeepAlive(store)
	return mkLinker(ptr, store)
}

func mkLinker(ptr *C.wasmtime_linker_t, store *Store) *Linker {
	linker := &Linker{_ptr: ptr, Store: store}
	runtime.SetFinalizer(linker, func(linker *Linker) {
		C.wasmtime_linker_delete(linker._ptr)
	})
	return linker
}

func (l *Linker) ptr() *C.wasmtime_linker_t {
	ret := l._ptr
	maybeGC()
	return ret
}

// Configures whether names can be redefined after they've already been defined
// in this linker.
func (l *Linker) AllowShadowing(allow bool) {
	C.wasmtime_linker_allow_shadowing(l.ptr(), C.bool(allow))
	runtime.KeepAlive(l)
}

// Defines a new item in this linker with the given module/name pair. Returns
// an error if shadowing is disallowed and the module/name is already defined.
func (l *Linker) Define(module, name string, item AsExtern) error {
	extern := item.AsExtern()
	ret := C.go_linker_define(
		l.ptr(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		C._GoStringPtr(name),
		C._GoStringLen(name),
		extern.ptr(),
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(name)
	runtime.KeepAlive(extern)
	if ret {
		return nil
	} else {
		return errors.New("failed to define item")
	}
}

// Convenience wrapper to calling Define and WrapFunc.
//
// Returns an error if shadowing is disabled and the name is already defined.
func (l *Linker) DefineFunc(module, name string, f interface{}) error {
	return l.Define(module, name, WrapFunc(l.Store, f))
}

// Defines all exports of an instance provided under the module name provided.
//
// Returns an error if shadowing is disabled and names are already defined.
func (l *Linker) DefineInstance(module string, instance *Instance) error {
	ret := C.go_linker_define_instance(
		l.ptr(),
		C._GoStringPtr(module),
		C._GoStringLen(module),
		instance.ptr(),
	)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	runtime.KeepAlive(instance)
	if ret {
		return nil
	} else {
		return errors.New("failed to define item")
	}
}

// Links a WASI module into this linker, ensuring that all exported functions
// are available for linking.
//
// Returns an error if shadowing is disabled and names are already defined.
func (l *Linker) DefineWasi(instance *WasiInstance) error {
	ret := C.wasmtime_linker_define_wasi(l.ptr(), instance.ptr())
	runtime.KeepAlive(l)
	runtime.KeepAlive(instance)
	if ret {
		return nil
	} else {
		return errors.New("failed to define item")
	}
}

// Instantates a module with all imports defined in this linker.
//
// Returns an error if the instance's imports couldn't be satisfied, had the
// wrong types, or if a trap happened executing the start function.
func (l *Linker) Instantiate(module *Module) (*Instance, error) {
	var trap *C.wasm_trap_t
	ret := C.wasmtime_linker_instantiate(l.ptr(), module.ptr(), &trap)
	runtime.KeepAlive(l)
	runtime.KeepAlive(module)
	if ret == nil {
		if trap != nil {
			return nil, mkTrap(trap)
		}
		return nil, errors.New("failed to instantiate")
	}
	return mkInstance(ret, module), nil
}
