package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/url"
)

//go:embed prelude.js
var prelude string

// installWebAPI adds web-standard globals that do no I/O, and returns the JS value formatter.
func installWebAPI(vm *goja.Runtime) func(goja.Value) string {
	// Enabling require would allow reading files without permission, so initialize the module directly.
	mod := vm.NewObject()
	exports := vm.NewObject()
	mod.Set("exports", exports)
	url.Require(vm, mod)
	vm.Set("URL", exports.Get("URL"))
	vm.Set("URLSearchParams", exports.Get("URLSearchParams"))

	host := map[string]any{
		"stdout":     func(s string) { fmt.Fprintln(os.Stdout, s) },
		"stderr":     func(s string) { fmt.Fprintln(os.Stderr, s) },
		"utf8Encode": func(s string) goja.ArrayBuffer { return vm.NewArrayBuffer([]byte(s)) },
		"utf8Decode": func(b []byte) string { return string(b) },
	}
	fn, _ := goja.AssertFunction(orPanic(vm.RunScript("prelude.js", prelude)))
	inspect, _ := goja.AssertFunction(orPanic(fn(goja.Undefined(), vm.ToValue(host))))
	return func(v goja.Value) string { return orPanic(inspect(goja.Undefined(), v)).String() }
}

// orPanic is for the embedded prelude, whose failure is a bug in eel.
func orPanic(v goja.Value, err error) goja.Value {
	if err != nil {
		panic(err)
	}
	return v
}
